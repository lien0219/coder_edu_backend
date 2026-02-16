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
	if ext != ".png" && ext != ".svg" {
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
	if _, err := util.ValidateMimeType(src, []string{"image/png", "image/svg+xml"}); err != nil {
		return "", fmt.Errorf("非法的文件内容，仅允许PNG或SVG格式: %v", err)
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
		ext := filepath.Ext(filename)
		videoID := util.GenerateRandomString(16)
		videoFilename := fmt.Sprintf("videos/%s%s", videoID, ext)
		finalPath := filepath.Join(s.Cfg.Storage.LocalPath, "temp", identifier+"_final"+ext)

		finalFile, err := os.Create(finalPath)
		if err != nil {
			return nil, nil, err
		}

		for i := 1; i <= totalChunks; i++ {
			chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
			f, err := os.Open(chunkPath)
			if err != nil {
				finalFile.Close()
				return nil, nil, err
			}
			_, err = io.Copy(finalFile, f)
			f.Close()
			if err != nil {
				finalFile.Close()
				return nil, nil, err
			}
		}
		finalFile.Close()

		// 上传合并后的文件
		finalURL, err := s.StorageService.UploadFile(ctx, videoFilename, finalPath, "video/"+strings.TrimPrefix(ext, "."))
		if err != nil {
			os.Remove(finalPath) // 失败也要清理
			return nil, nil, err
		}

		// 如果没有提供标题，使用文件名
		if title == "" {
			title = strings.TrimSuffix(filename, ext)
		}

		// 1. 同步获取元数据（分片上传最后一步需要准确的时长和封面）
		duration, thumbnail := s.processVideoMetadata(ctx, finalURL, finalPath, filename)

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
			s.StorageService.Delete(ctx, videoFilename) // 清理孤立文件
			os.Remove(finalPath)
			return nil, nil, err
		}

		// 2. 异步执行清理工作
		s.wg.Add(1)
		go func(lPath, tDir, rKey string) {
			defer s.wg.Done()
			// 延迟几秒清理，确保 localPath 不再被读取（如果 FFmpeg 还没关的话）
			time.Sleep(2 * time.Second)
			os.Remove(lPath)
			os.RemoveAll(tDir)
			s.Redis.Del(context.Background(), rKey)
		}(finalPath, tempDir, redisKey)

		return progress, resource, nil
	}

	return progress, nil, nil
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
		thumbnailURL = videoURL + "?x-oss-process=video/snapshot,t_7000,f_jpg,w_800"
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

// getVideoDurationFromOSS 从阿里云OSS获取视频时长（带重试逻辑，解决IMM索引延迟）
func (s *ContentService) getVideoDurationFromOSS(videoURL string) float64 {
	u, err := url.Parse(videoURL)
	if err != nil {
		logger.Log.Error("解析视频URL失败", zap.Error(err))
		return 0
	}

	infoURL := fmt.Sprintf("%s://%s%s?x-oss-process=video/info", u.Scheme, u.Host, u.EscapedPath())

	// 增加重试逻辑，采用指数退避策略
	backoff := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	for i := 0; i < len(backoff); i++ {
		if i > 0 {
			time.Sleep(backoff[i-1])
		}

		resp, err := s.httpClient.Get(infoURL)
		if err != nil {
			logger.Log.Error("请求OSS时长接口失败", zap.Error(err), zap.Int("retry", i))
			continue
		}

		// 如果返回非 200，说明索引还没好或未授权，继续重试
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			logger.Log.Warn("OSS视频信息未就绪，重试中...",
				zap.Int("status", resp.StatusCode),
				zap.Int("retry", i),
				zap.String("response", string(body)))
			continue
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			logger.Log.Error("解析OSS视频信息JSON失败", zap.Error(err))
			return 0
		}
		resp.Body.Close()

		// 增强解析逻辑：检查多个可能的层级
		if format, ok := result["format"].(map[string]interface{}); ok {
			if d, exists := format["duration"]; exists {
				durationStr := fmt.Sprintf("%v", d)
				duration, _ := strconv.ParseFloat(durationStr, 64)
				if duration > 0 {
					return duration
				}
			}
		}

		// 2. streams[0].duration (兼容某些格式)
		if streams, ok := result["streams"].([]interface{}); ok && len(streams) > 0 {
			if firstStream, ok := streams[0].(map[string]interface{}); ok {
				if d, exists := firstStream["duration"]; exists {
					durationStr := fmt.Sprintf("%v", d)
					duration, _ := strconv.ParseFloat(durationStr, 64)
					if duration > 0 {
						return duration
					}
				}
			}
		}
	}

	return 0
}
