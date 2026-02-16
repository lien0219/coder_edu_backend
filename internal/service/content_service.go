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
	"os"
	"path/filepath"
	"strings"
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
}

func NewContentService(resourceRepo *repository.ResourceRepository, storageService *StorageService, cfg *config.Config, rdb *redis.Client) *ContentService {
	return &ContentService{
		ResourceRepo:   resourceRepo,
		StorageService: storageService,
		Cfg:            cfg,
		Redis:          rdb,
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

	// 使用当前时间生成唯一文件名
	videoFilename := "videos/" + time.Now().Format("20060102150405") + "-" +
		strings.ReplaceAll(file.Filename, " ", "-")

	// 临时保存到本地进行处理
	tempDir := filepath.Join(s.Cfg.Storage.LocalPath, "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, err
	}

	tempFilename := fmt.Sprintf("temp_video_%d%s", time.Now().UnixNano(), ext)
	videoPath := filepath.Join(tempDir, tempFilename)
	defer os.Remove(videoPath) // 上传完成后立即清理

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

	// 生成缩略图
	thumbnailExt := ".jpg"
	thumbnailFilename := "thumbnails/" + time.Now().Format("20060102150405") + "-" +
		strings.ReplaceAll(strings.TrimSuffix(file.Filename, ext), " ", "-") + thumbnailExt

	thumbnailDir := filepath.Join(s.Cfg.Storage.LocalPath, "thumbnails")
	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return nil, err
	}
	thumbnailPath := filepath.Join(thumbnailDir, filepath.Base(thumbnailFilename))
	defer os.Remove(thumbnailPath) // 处理完成后清理

	var thumbnailURL string
	err = util.GenerateThumbnail(videoPath, thumbnailPath, "3")
	if err != nil {
		logger.Log.Error("生成缩略图失败", zap.Error(err))
		thumbnailURL = s.StorageService.GetURL("thumbnails/default-video-thumbnail.jpg")
	} else {
		thumbnailURL, err = s.StorageService.UploadFile(ctx, thumbnailFilename, thumbnailPath, "image/jpeg")
		if err != nil {
			thumbnailURL = s.StorageService.GetURL("thumbnails/default-video-thumbnail.jpg")
		}
	}

	// 获取视频时长
	videoInfo, err := util.GetVideoInfo(videoPath)
	var duration float64 = 0
	if err == nil {
		duration = videoInfo.Duration
	}

	resource := &model.Resource{
		Title:       title,
		Description: description,
		Type:        model.Video,
		URL:         videoURL,
		Duration:    duration,
		Size:        file.Size,
		Format:      strings.TrimPrefix(ext, "."),
		Thumbnail:   thumbnailURL,
	}

	if err := s.ResourceRepo.Create(resource); err != nil {
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
		videoFilename := "videos/" + time.Now().Format("20060102150405") + "-" +
			strings.ReplaceAll(strings.TrimSuffix(filename, ext), " ", "-") + ext
		finalPath := filepath.Join(s.Cfg.Storage.LocalPath, "temp", identifier+"_final"+ext)

		finalFile, err := os.Create(finalPath)
		if err != nil {
			return nil, nil, err
		}

		for i := 1; i <= totalChunks; i++ {
			chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
			data, err := os.ReadFile(chunkPath)
			if err != nil {
				finalFile.Close()
				return nil, nil, err
			}
			if _, err := finalFile.Write(data); err != nil {
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

		// 生成缩略图
		thumbnailExt := ".jpg"
		thumbnailFilename := "thumbnails/" + time.Now().Format("20060102150405") + "-" +
			util.GenerateRandomString(6) + thumbnailExt

		thumbnailDir := filepath.Join(s.Cfg.Storage.LocalPath, "thumbnails")
		if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
			logger.Log.Error("创建缩略图目录失败", zap.Error(err))
		}
		thumbnailPath := filepath.Join(thumbnailDir, filepath.Base(thumbnailFilename))

		var thumbnailURL string
		err = util.GenerateThumbnail(finalPath, thumbnailPath, "3")
		if err != nil {
			logger.Log.Error("生成缩略图失败", zap.Error(err))
			thumbnailURL = s.StorageService.GetURL("thumbnails/default-video-thumbnail.jpg")
		} else {
			thumbnailURL, err = s.StorageService.UploadFile(ctx, thumbnailFilename, thumbnailPath, "image/jpeg")
			if err != nil {
				thumbnailURL = s.StorageService.GetURL("thumbnails/default-video-thumbnail.jpg")
			}
			os.Remove(thumbnailPath) // 上传后清理本地临时封面
		}

		// 获取视频时长
		videoInfo, err := util.GetVideoInfo(finalPath)
		var duration float64 = 0
		if err == nil {
			duration = videoInfo.Duration
		}

		// 如果没有提供标题，使用文件名
		if title == "" {
			title = strings.TrimSuffix(filename, ext)
		}

		// 创建资源记录
		resource = &model.Resource{
			Title:       title,
			Description: description,
			Type:        model.Video,
			URL:         finalURL,
			Duration:    duration,
			Size:        progress.FileSize,
			Format:      strings.TrimPrefix(ext, "."),
			Thumbnail:   thumbnailURL,
		}

		if err := s.ResourceRepo.Create(resource); err != nil {
			logger.Log.Error("创建资源记录失败", zap.Error(err))
		}

		// 清理
		os.RemoveAll(tempDir)
		os.Remove(finalPath)
		// 从Redis中删除进度
		s.Redis.Del(context.Background(), redisKey)
	}

	return progress, resource, nil
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
