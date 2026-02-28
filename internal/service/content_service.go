package service

import (
	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"coder_edu_backend/internal/util"
	"coder_edu_backend/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type ContentService struct {
	ResourceRepo   *repository.ResourceRepository
	StorageService *StorageService
	Cfg            *config.Config
	Redis          *redis.Client
	httpClient     *http.Client
	workerSem      chan struct{}  // 并发控制信号量
	wg             sync.WaitGroup // 优雅停机等待组
}

func NewContentService(resourceRepo *repository.ResourceRepository, storageService *StorageService, cfg *config.Config, rdb *redis.Client) *ContentService {
	return &ContentService{
		ResourceRepo:   resourceRepo,
		StorageService: storageService,
		Cfg:            cfg,
		Redis:          rdb,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				MaxIdleConnsPerHost: 20,
			},
		},
		workerSem: make(chan struct{}, 5), // 限制同时处理视频数量
	}
}

// Shutdown 优雅停机，等待所有后台任务完成
func (s *ContentService) Shutdown(ctx context.Context) error {
	c := make(chan struct{})
	go func() {
		defer close(c)
		s.wg.Wait()
	}()

	select {
	case <-c:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

const uploadProgressKeyPrefix = "upload_progress:"

func (s *ContentService) UploadResource(c *gin.Context, file *multipart.FileHeader, resource *model.Resource) error {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		return util.ErrUnauthorized
	}

	resource.UploaderID = claims.UserID

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// 深度验证 MIME 类型
	allowedTypes := []string{util.MimePDF, util.MimeVideo, util.MimeImage, "text/plain", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"}
	if _, err := util.ValidateMimeType(src, allowedTypes); err != nil {
		return fmt.Errorf("非法的文件内容: %v", err)
	}
	// 重置读取指针
	if seeker, ok := src.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	ext := filepath.Ext(file.Filename)
	filename := "resources/" + time.Now().Format("20060102150405") + "_" + util.GenerateRandomString(6) + ext

	url, err := s.StorageService.Upload(c, filename, src, file.Size, file.Header.Get("Content-Type"))
	if err != nil {
		return err
	}

	resource.URL = url
	return s.ResourceRepo.Create(resource)
}

func (s *ContentService) UploadIcon(ctx context.Context, file *multipart.FileHeader) (string, error) {
	// 验证文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".png" && ext != ".svg" && ext != ".jpg" && ext != ".jpeg" {
		return "", util.ErrInvalidIconExt
	}

	// 使用当前时间生成唯一文件名
	filename := "icons/" + time.Now().Format("20060102150405") + "-" +
		strings.ReplaceAll(file.Filename, " ", "-")

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// 深度验证 MIME 类型
	if _, err := util.ValidateMimeType(src, []string{"image/png", "image/svg+xml", "image/jpeg"}); err != nil {
		return "", fmt.Errorf("非法的文件内容，仅允许PNG、JPG或SVG格式: %v", err)
	}
	// 重置读取指针
	if seeker, ok := src.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	return s.StorageService.Upload(ctx, filename, src, file.Size, file.Header.Get("Content-Type"))
}

func (s *ContentService) UploadVideo(ctx context.Context, file *multipart.FileHeader, title, description string) (*model.Resource, error) {
	// 验证文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validVideoExts := util.AllowedVideoExtensions
	isValidType := false
	for _, e := range validVideoExts {
		if ext == e {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return nil, util.ErrInvalidVideoExt
	}

	videoID := util.GenerateRandomString(16)
	videoFilename := fmt.Sprintf("videos/%s%s", videoID, ext)

	// 临时保存到本地进行处理
	tempDir := filepath.Join(s.Cfg.Storage.LocalPath, "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, err
	}

	tempFilename := fmt.Sprintf("temp_video_%d%s", time.Now().UnixNano(), ext)
	videoPath := filepath.Join(tempDir, tempFilename)
	defer os.Remove(videoPath)

	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// 深度验证 MIME 类型
	if _, err := util.ValidateMimeType(src, []string{util.MimeVideo}); err != nil {
		return nil, fmt.Errorf("非法的文件内容，仅允许视频格式: %v", err)
	}
	// 重置读取指针
	if seeker, ok := src.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	dst, err := os.Create(videoPath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return nil, err
	}

	// 上传视频
	videoURL, err := s.StorageService.UploadFile(ctx, videoFilename, videoPath, file.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	// 同步获取元数据，确保返回给前端正确的数据
	duration, thumbnailURL := s.processVideoMetadata(ctx, videoURL, videoPath, file.Filename)

	resource := &model.Resource{
		Title:       title,
		Description: description,
		Type:        model.Video,
		Status:      model.ResourceSuccess,
		URL:         videoURL,
		Duration:    duration,
		Size:        file.Size,
		Format:      strings.TrimPrefix(ext, "."),
		Thumbnail:   thumbnailURL,
	}

	if err := s.ResourceRepo.Create(resource); err != nil {
		s.StorageService.Delete(ctx, videoFilename)
		return nil, err
	}

	return resource, nil
}

func (s *ContentService) UploadVideoChunk(ctx context.Context, chunkFile *multipart.FileHeader, chunkNumber, totalChunks int, identifier, filename string, title, description string) (*model.UploadProgress, *model.Resource, error) {
	// 创建临时目录存储分块
	tempDir := filepath.Join(s.Cfg.Storage.LocalPath, "temp", identifier)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, nil, err
	}

	// 保存分块文件
	chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", chunkNumber))
	src, err := chunkFile.Open()
	if err != nil {
		return nil, nil, err
	}
	defer src.Close()

	dst, err := os.Create(chunkPath)
	if err != nil {
		return nil, nil, err
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return nil, nil, err
	}
	dst.Close() // 写入完成后立即关闭，不要等 defer，防止win文件锁问题

	// 更新进度 (使用Redis----方便共享)
	redisKey := uploadProgressKeyPrefix + identifier
	var progress *model.UploadProgress

	// 获取现有进度
	val, err := s.Redis.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		progress = &model.UploadProgress{
			TotalChunks:    totalChunks,
			UploadedChunks: 0,
			FileSize:       0,
			Identifier:     identifier,
			Filename:       filename,
			CreatedAt:      time.Now(),
			Chunks:         make(map[int]bool),
		}
	} else if err != nil {
		return nil, nil, err
	} else {
		if err := json.Unmarshal([]byte(val), &progress); err != nil {
			return nil, nil, err
		}
		if progress.Chunks == nil {
			progress.Chunks = make(map[int]bool)
		}
	}

	// 更新进度
	if !progress.Chunks[chunkNumber] {
		progress.UploadedChunks++
		progress.FileSize += chunkFile.Size
		progress.Chunks[chunkNumber] = true
	}

	isComplete := progress.UploadedChunks == progress.TotalChunks

	// 保存回Redis(设置24小时过期)
	updatedVal, _ := json.Marshal(progress)
	if err := s.Redis.Set(ctx, redisKey, updatedVal, 24*time.Hour).Err(); err != nil {
		return nil, nil, err
	}

	var resource *model.Resource
	if isComplete {
		startTime := time.Now()
		ext := filepath.Ext(filename)
		videoID := util.GenerateRandomString(16)
		videoFilename := fmt.Sprintf("videos/%s%s", videoID, ext)
		var finalURL string
		var finalPath string
		var err error

		if s.Cfg.Storage.Type == util.StorageOSS {
			// 构建分块路径列表
			chunkPaths := make([]string, totalChunks)
			for i := 1; i <= totalChunks; i++ {
				chunkPaths[i-1] = filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
			}

			uploadStartTime := time.Now()
			finalURL, err = s.StorageService.UploadChunks(ctx, videoFilename, chunkPaths, "video/"+strings.TrimPrefix(ext, "."))
			uploadDuration := time.Since(uploadStartTime)
			if err != nil {
				return nil, nil, fmt.Errorf("OSS分片上传失败: %w", err)
			}
			logger.Log.Info("OSS分片上传完成（跳过合并步骤）",
				zap.String("identifier", identifier),
				zap.String("filename", videoFilename),
				zap.Int64("size", progress.FileSize),
				zap.Int("chunks", totalChunks),
				zap.Duration("duration", uploadDuration),
				zap.Float64("speed_mbps", float64(progress.FileSize)/(1024*1024)/uploadDuration.Seconds()))

			finalPath = ""
		} else {
			// 非OSS类型，需要先合并再上传
			finalPath = filepath.Join(s.Cfg.Storage.LocalPath, "temp", identifier+"_final"+ext)
			if err := s.mergeChunksConcurrently(tempDir, finalPath, totalChunks); err != nil {
				return nil, nil, fmt.Errorf("合并分块失败: %w", err)
			}
			mergeDuration := time.Since(startTime)
			logger.Log.Info("文件合并完成", zap.String("identifier", identifier), zap.Int("chunks", totalChunks), zap.Duration("duration", mergeDuration))

			// 上传合并后的文件
			uploadStartTime := time.Now()
			finalURL, err = s.StorageService.UploadFile(ctx, videoFilename, finalPath, "video/"+strings.TrimPrefix(ext, "."))
			uploadDuration := time.Since(uploadStartTime)
			if err != nil {
				os.Remove(finalPath)
				return nil, nil, err
			}
			logger.Log.Info("文件上传到存储服务完成",
				zap.String("identifier", identifier),
				zap.String("filename", videoFilename),
				zap.Int64("size", progress.FileSize),
				zap.Duration("duration", uploadDuration),
				zap.Float64("speed_mbps", float64(progress.FileSize)/(1024*1024)/uploadDuration.Seconds()))
		}

		// 如果没有提供标题，使用文件名
		if title == "" {
			title = strings.TrimSuffix(filename, ext)
		}

		// 1. 等待元数据就绪（确保返回给前端的数据完整）
		metadataStartTime := time.Now()
		localPathForMetadata := ""
		if s.Cfg.Storage.Type == util.StorageOSS {
			localPathForMetadata = ""
		} else {
			localPathForMetadata = finalPath
		}
		duration, thumbnail := s.processVideoMetadataWithWait(ctx, finalURL, localPathForMetadata, filename, tempDir, totalChunks, s.Cfg.Storage.Type == util.StorageOSS)
		metadataDuration := time.Since(metadataStartTime)
		logger.Log.Info("视频元数据处理完成",
			zap.String("identifier", identifier),
			zap.Float64("duration", duration),
			zap.String("thumbnail", thumbnail),
			zap.Duration("process_time", metadataDuration))

		resource = &model.Resource{
			Title:       title,
			Description: description,
			Type:        model.Video,
			Status:      model.ResourceSuccess,
			URL:         finalURL,
			Duration:    duration,
			Size:        progress.FileSize,
			Format:      strings.TrimPrefix(ext, "."),
			Thumbnail:   thumbnail,
		}

		if err := s.ResourceRepo.Create(resource); err != nil {
			logger.Log.Error("创建资源记录失败", zap.Error(err))
			s.StorageService.Delete(ctx, videoFilename)
			if finalPath != "" {
				os.Remove(finalPath)
			}
			return nil, nil, err
		}

		if duration == 0 {
			s.wg.Add(1)
			go func(vURL, lPath, fName string, resID uint) {
				defer s.wg.Done()
				// 后台详细处理元数据
				dur, thumb := s.processVideoMetadata(context.Background(), vURL, lPath, fName)
				if dur > 0 || thumb != "" {
					updates := make(map[string]interface{})
					if dur > 0 {
						updates["duration"] = dur
					}
					if thumb != "" {
						updates["thumbnail"] = thumb
					}
					s.ResourceRepo.UpdateFields(resID, model.Video, updates)
					logger.Log.Info("后台更新视频元数据完成", zap.Uint("resource_id", resID))
				}
			}(finalURL, finalPath, filename, resource.ID)
		}

		// 2. 异步执行清理工作
		// 注意：需要等待足够长的时间，确保 processVideoMetadataWithWait 中的FFmpeg处理完成
		// FFmpeg处理可能需要临时合并文件，所以延迟时间要足够长（至少60秒，因为OSS超时是60秒）
		s.wg.Add(1)
		go func(lPath, tDir, rKey string, isOSS bool) {
			defer s.wg.Done()
			// 延迟足够长的时间，确保元数据处理完成（包括FFmpeg备选方案）
			// OSS超时60秒 + FFmpeg处理时间，总共延迟70秒
			logger.Log.Info("开始等待清理临时文件",
				zap.String("tempDir", tDir),
				zap.String("localPath", lPath),
				zap.Duration("delay", 70*time.Second))
			time.Sleep(70 * time.Second)

			// 清理临时文件
			if lPath != "" {
				if err := os.Remove(lPath); err != nil {
					logger.Log.Warn("清理临时合并文件失败", zap.String("path", lPath), zap.Error(err))
				} else {
					logger.Log.Info("已清理临时合并文件", zap.String("path", lPath))
				}
			}

			if err := os.RemoveAll(tDir); err != nil {
				logger.Log.Warn("清理临时分块目录失败", zap.String("dir", tDir), zap.Error(err))
			} else {
				logger.Log.Info("已清理临时分块目录", zap.String("dir", tDir))
			}

			if err := s.Redis.Del(context.Background(), rKey).Err(); err != nil {
				logger.Log.Warn("清理Redis进度记录失败", zap.String("key", rKey), zap.Error(err))
			} else {
				logger.Log.Info("已清理Redis进度记录", zap.String("key", rKey))
			}
		}(finalPath, tempDir, redisKey, s.Cfg.Storage.Type == util.StorageOSS)

		return progress, resource, nil
	}

	return progress, nil, nil
}

// mergeChunksConcurrently 并发合并分块文件，提高大文件合并速度
func (s *ContentService) mergeChunksConcurrently(tempDir, finalPath string, totalChunks int) error {
	finalFile, err := os.Create(finalPath)
	if err != nil {
		return err
	}
	defer finalFile.Close()

	// 对于小文件（少于10个分块），使用顺序合并
	if totalChunks <= 10 {
		for i := 1; i <= totalChunks; i++ {
			chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
			f, err := os.Open(chunkPath)
			if err != nil {
				return err
			}
			_, err = io.Copy(finalFile, f)
			f.Close()
			if err != nil {
				return err
			}
		}
		return nil
	}

	// 流式写入，避免大文件全部读入内存
	buf := make([]byte, 32*1024) // 32KB 缓冲区
	for i := 1; i <= totalChunks; i++ {
		chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("打开分块 %d 失败: %w", i, err)
		}

		// 使用缓冲区流式复制
		_, err = io.CopyBuffer(finalFile, chunkFile, buf)
		chunkFile.Close()
		if err != nil {
			return fmt.Errorf("写入分块 %d 失败: %w", i, err)
		}
	}

	return nil
}

func (s *ContentService) GetUploadProgress(identifier string) (*model.UploadProgress, error) {
	redisKey := uploadProgressKeyPrefix + identifier
	val, err := s.Redis.Get(context.Background(), redisKey).Result()
	if err == redis.Nil {
		return nil, util.ErrUploadProgressNotFound
	} else if err != nil {
		return nil, err
	}

	var progress model.UploadProgress
	if err := json.Unmarshal([]byte(val), &progress); err != nil {
		return nil, err
	}
	return &progress, nil
}

func (s *ContentService) UpdateResource(id uint, resourceType model.ResourceType, updates map[string]interface{}) error {
	return s.ResourceRepo.UpdateFields(id, resourceType, updates)
}

func (s *ContentService) DeleteResource(id uint, resourceType model.ResourceType) error {
	return s.ResourceRepo.DeleteByType(id, resourceType)
}

// processVideoMetadataWithWait 等待元数据就绪（并发获取，确保数据完整）
// localPath: 已合并的文件路径（如果为空，表示需要时再合并）
// tempDir: 临时分块目录（用于延迟合并）
// totalChunks: 分块总数
// isOSS: 是否为OSS类型
func (s *ContentService) processVideoMetadataWithWait(ctx context.Context, videoURL, localPath, originalFilename, tempDir string, totalChunks int, isOSS bool) (float64, string) {
	type durationResult struct {
		duration float64
	}
	type thumbnailResult struct {
		thumbnail string
	}

	durationChan := make(chan durationResult, 1)
	thumbnailChan := make(chan thumbnailResult, 1)

	// 并发获取时长和封面
	go func() {
		var duration float64

		// 对于OSS类型，优先使用FFmpeg（因为分块文件还在，可以立即合并获取）
		// 这样可以避免等待OSS视频处理服务（可能需要几十秒）
		if isOSS && tempDir != "" {
			// OSS类型，优先使用FFmpeg（分块文件还在）
			logger.Log.Info("OSS类型，优先使用FFmpeg获取视频时长（分块文件可用）", zap.String("videoURL", videoURL))
			ext := filepath.Ext(originalFilename)
			actualLocalPath := filepath.Join(s.Cfg.Storage.LocalPath, "temp", fmt.Sprintf("temp_metadata_%d%s", time.Now().UnixNano(), ext))
			mergeStartTime := time.Now()
			if err := s.mergeChunksConcurrently(tempDir, actualLocalPath, totalChunks); err == nil {
				// 合并成功，使用合并后的文件
				defer os.Remove(actualLocalPath) // 处理完后清理
				logger.Log.Info("临时合并文件成功，用于FFmpeg处理",
					zap.String("path", actualLocalPath),
					zap.Duration("merge_time", time.Since(mergeStartTime)))

				ffmpegStartTime := time.Now()
				if videoInfo, err := util.GetVideoInfo(actualLocalPath); err == nil {
					duration = videoInfo.Duration
					logger.Log.Info("FFmpeg成功获取视频时长",
						zap.Float64("duration", duration),
						zap.Duration("ffmpeg_time", time.Since(ffmpegStartTime)))
				} else {
					logger.Log.Warn("FFmpeg获取视频时长失败（可能未安装FFmpeg），将尝试OSS", zap.Error(err))
				}
			} else {
				logger.Log.Warn("临时合并文件失败，无法使用FFmpeg，将尝试OSS", zap.Error(err))
			}
		} else if localPath != "" {
			// 非OSS类型，直接使用已合并的文件
			logger.Log.Info("使用本地文件获取视频时长", zap.String("path", localPath))
			if videoInfo, err := util.GetVideoInfo(localPath); err == nil {
				duration = videoInfo.Duration
			}
		}

		// 如果FFmpeg/本地文件获取失败，尝试从OSS获取（作为备选方案）
		if duration == 0 && s.Cfg.Storage.Type == util.StorageOSS {
			logger.Log.Info("FFmpeg/本地文件获取失败，尝试从OSS获取视频时长", zap.String("videoURL", videoURL))
			duration = s.getVideoDurationFromOSS(videoURL)
		}

		durationChan <- durationResult{duration: duration}
	}()

	go func() {
		var thumbnailURL string
		if s.Cfg.Storage.Type == util.StorageOSS {
			publicVideoURL := strings.Replace(videoURL, "-internal", "", 1)
			thumbnailURL = publicVideoURL + "?x-oss-process=video/snapshot,t_7000,f_jpg,w_800"
		}

		actualLocalPath := localPath
		if thumbnailURL == "" && localPath == "" && isOSS && tempDir != "" {
			// 临时合并文件用于FFmpeg处理
			ext := filepath.Ext(originalFilename)
			actualLocalPath = filepath.Join(s.Cfg.Storage.LocalPath, "temp", fmt.Sprintf("temp_thumbnail_%d%s", time.Now().UnixNano(), ext))
			if err := s.mergeChunksConcurrently(tempDir, actualLocalPath, totalChunks); err == nil {
				defer os.Remove(actualLocalPath) // 处理完后清理
			} else {
				actualLocalPath = "" // 合并失败，无法使用
			}
		}
		if thumbnailURL == "" && actualLocalPath != "" {
			thumbnailExt := ".jpg"
			thumbnailFilename := "thumbnails/" + time.Now().Format("20060102150405") + "-" +
				util.GenerateRandomString(6) + thumbnailExt

			thumbnailDir := filepath.Join(s.Cfg.Storage.LocalPath, "thumbnails")
			os.MkdirAll(thumbnailDir, 0755)
			thumbnailPath := filepath.Join(thumbnailDir, filepath.Base(thumbnailFilename))

			if err := util.GenerateThumbnail(localPath, thumbnailPath, "3"); err == nil {
				uploadedURL, err := s.StorageService.UploadFile(ctx, thumbnailFilename, thumbnailPath, "image/jpeg")
				if err == nil {
					thumbnailURL = uploadedURL
				}
				os.Remove(thumbnailPath)
			}
		}

		// 如果所有方案都失败，使用默认占位图
		if thumbnailURL == "" {
			thumbnailURL = s.StorageService.GetURL("thumbnails/default-video-thumbnail.jpg")
		}
		thumbnailChan <- thumbnailResult{thumbnail: thumbnailURL}
	}()

	// 等待两个结果都返回
	durationRes := <-durationChan
	thumbnailRes := <-thumbnailChan

	return durationRes.duration, thumbnailRes.thumbnail
}

// processVideoMetadata 处理视频元数据（时长和封面）
func (s *ContentService) processVideoMetadata(ctx context.Context, videoURL, localPath, originalFilename string) (float64, string) {
	// 1. 获取视频时长
	var duration float64
	if s.Cfg.Storage.Type == util.StorageOSS {
		duration = s.getVideoDurationFromOSS(videoURL)
	}

	// 如果不是 OSS 或 OSS 获取失败，尝试使用本地 FFmpeg
	if duration == 0 && localPath != "" {
		if videoInfo, err := util.GetVideoInfo(localPath); err == nil {
			duration = videoInfo.Duration
		}
	}

	// 2. 生成封面图
	var thumbnailURL string
	if s.Cfg.Storage.Type == util.StorageOSS {
		publicVideoURL := strings.Replace(videoURL, "-internal", "", 1)
		thumbnailURL = publicVideoURL + "?x-oss-process=video/snapshot,t_7000,f_jpg,w_800"
	} else if localPath != "" {
		thumbnailExt := ".jpg"
		thumbnailFilename := "thumbnails/" + time.Now().Format("20060102150405") + "-" +
			util.GenerateRandomString(6) + thumbnailExt

		thumbnailDir := filepath.Join(s.Cfg.Storage.LocalPath, "thumbnails")
		os.MkdirAll(thumbnailDir, 0755)
		thumbnailPath := filepath.Join(thumbnailDir, filepath.Base(thumbnailFilename))

		if err := util.GenerateThumbnail(localPath, thumbnailPath, "3"); err == nil {
			thumbnailURL, _ = s.StorageService.UploadFile(ctx, thumbnailFilename, thumbnailPath, "image/jpeg")
			os.Remove(thumbnailPath)
		}
	}

	// 如果所有截图方案都失败，使用默认占位图
	if thumbnailURL == "" {
		thumbnailURL = s.StorageService.GetURL("thumbnails/default-video-thumbnail.jpg")
	}

	return duration, thumbnailURL
}

// getVideoDurationFromOSS 从阿里云OSS获取视频时长
// 总超时时间60秒，如果OSS获取失败会尝试使用FFmpeg备选方案
func (s *ContentService) getVideoDurationFromOSS(videoURL string) float64 {
	u, err := url.Parse(videoURL)
	if err != nil {
		logger.Log.Error("解析视频URL失败", zap.Error(err))
		return 0
	}

	host := strings.Replace(u.Host, "-internal", "", 1)
	infoURL := fmt.Sprintf("%s://%s%s?x-oss-process=video/info", u.Scheme, host, u.EscapedPath())

	// 设置总超时时间为60秒
	timeout := 60 * time.Second
	startTime := time.Now()

	// 初始重试间隔
	backoff := 2 * time.Second     // 初始间隔2秒
	maxBackoff := 10 * time.Second // 最大间隔10秒
	retryCount := 0

	for {
		// 检查是否超时
		if time.Since(startTime) >= timeout {
			logger.Log.Warn("获取OSS视频时长超时", zap.Duration("timeout", timeout), zap.Int("retries", retryCount))
			break
		}

		// 如果不是第一次请求，等待退避时间
		if retryCount > 0 {
			time.Sleep(backoff)
			// 指数退避，但不超过最大间隔
			backoff = time.Duration(float64(backoff) * 1.5)
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}

		resp, err := s.httpClient.Get(infoURL)
		retryCount++

		if err != nil {
			logger.Log.Warn("请求OSS时长接口失败，重试中...", zap.Error(err), zap.Int("retry", retryCount), zap.Duration("elapsed", time.Since(startTime)))
			continue
		}

		// 如果返回非 200，说明索引还没好或未授权，继续重试
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if retryCount%10 == 0 {
				logger.Log.Warn("OSS视频信息未就绪，持续重试中...",
					zap.Int("status", resp.StatusCode),
					zap.Int("retry", retryCount),
					zap.Duration("elapsed", time.Since(startTime)),
					zap.String("response", string(body)))
			}
			continue
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			logger.Log.Warn("解析OSS视频信息JSON失败，重试中...", zap.Error(err), zap.Int("retry", retryCount))
			continue
		}
		resp.Body.Close()

		var duration float64
		if format, ok := result["format"].(map[string]interface{}); ok {
			if d, exists := format["duration"]; exists {
				durationStr := fmt.Sprintf("%v", d)
				duration, _ = strconv.ParseFloat(durationStr, 64)
				if duration > 0 {
					logger.Log.Info("成功获取OSS视频时长", zap.Float64("duration", duration), zap.Int("retries", retryCount))
					return duration
				}
			}
		}

		// 2. streams[0].duration (兼容某些格式)
		if streams, ok := result["streams"].([]interface{}); ok && len(streams) > 0 {
			if firstStream, ok := streams[0].(map[string]interface{}); ok {
				if d, exists := firstStream["duration"]; exists {
					durationStr := fmt.Sprintf("%v", d)
					duration, _ = strconv.ParseFloat(durationStr, 64)
					if duration > 0 {
						logger.Log.Info("成功获取OSS视频时长", zap.Float64("duration", duration), zap.Int("retries", retryCount))
						return duration
					}
				}
			}
		}

		// 如果解析成功但没有找到duration，继续重试
		logger.Log.Warn("OSS返回数据中未找到duration字段，重试中...", zap.Int("retry", retryCount))
	}

	logger.Log.Warn("获取OSS视频时长失败",
		zap.Int("total_retries", retryCount),
		zap.Duration("elapsed", time.Since(startTime)))
	return 0
}
