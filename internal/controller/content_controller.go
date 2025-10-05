package controller

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/internal/util"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type ContentController struct {
	ContentService *service.ContentService
}

func NewContentController(contentService *service.ContentService) *ContentController {
	return &ContentController{ContentService: contentService}
}

// UploadResourceRequest defines model for resource upload
// swagger:model UploadResourceRequest
type UploadResourceRequest struct {
	Title       string `form:"title" binding:"required"`
	Description string `form:"description"`
	Type        string `form:"type" binding:"required,oneof=pdf video article worksheet"`
	ModuleType  string `form:"moduleType" binding:"required,oneof=pre-class in-class post-class"`
}

// UploadResource godoc
// @Summary 上传学习资源（仅管理员）
// @Description 上传PDF、视频、文章或工作表到特定模块
// @Tags 内容
// @Accept  multipart/form-data
// @Produce  json
// @Security ApiKeyAuth
// @Param   title formData string true "资源标题"
// @Param   description formData string false "资源描述"
// @Param   type formData string true "资源类型（pdf, video, article, worksheet)" Enums(pdf, video, article, worksheet)
// @Param   moduleType formData string true "模块类型（pre-class, in-class, post-class)" Enums(pre-class, in-class, post-class)
// @Param   file formData file true "资源文件"
// @Success 201 {object} util.Response{data=object} "创建成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/admin/resources [post]
func (c *ContentController) UploadResource(ctx *gin.Context) {
	var req UploadResourceRequest
	if err := ctx.ShouldBind(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		util.BadRequest(ctx, "File is required")
		return
	}

	resource := &model.Resource{
		Title:       req.Title,
		Description: req.Description,
		Type:        model.ResourceType(req.Type),
		ModuleType:  req.ModuleType,
	}

	if err := c.ContentService.UploadResource(ctx, file, resource); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Created(ctx, gin.H{"id": resource.ID, "url": resource.URL})
}

// GetResources godoc
// @Summary 按模块类型获取资源
// @Description 获取特定模块类型的资源列表
// @Tags 内容
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   moduleType query string true "模块类型（pre-class, in-class, post-class)" Enums(pre-class, in-class, post-class)
// @Success 200 {object} util.Response{data=[]model.Resource} "成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/resources [get]
func (c *ContentController) GetResources(ctx *gin.Context) {
	moduleType := ctx.Query("moduleType")
	if moduleType == "" {
		util.BadRequest(ctx, "moduleType parameter is required")
		return
	}

	resources, err := c.ContentService.ResourceRepo.FindByModule(moduleType)
	if err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, resources)
}

// UploadIcon godoc
// @Summary 上传模块图标（仅管理员）
// @Description 专门用于上传C语言编程模块的图标
// @Tags 内容
// @Accept  multipart/form-data
// @Produce  json
// @Security ApiKeyAuth
// @Param   icon formData file true "图标文件（PNG或SVG格式）"
// @Success 200 {object} util.Response{data=map[string]string} "上传成功"
// @Failure 400 {object} util.Response "请求参数错误或文件格式不支持"
// @Failure 401 {object} util.Response "未授权"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/admin/upload/icon [post]
func (c *ContentController) UploadIcon(ctx *gin.Context) {
	file, err := ctx.FormFile("icon")
	if err != nil {
		util.BadRequest(ctx, "图标文件是必需的")
		return
	}

	// 验证文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".png" && ext != ".svg" {
		util.BadRequest(ctx, "文件格式不支持，请上传PNG或SVG格式")
		return
	}

	// 使用当前时间生成唯一文件名
	filename := "icons/" + time.Now().Format("20060102150405") + "-" +
		strings.ReplaceAll(file.Filename, " ", "-")

	// 根据存储类型进行上传
	var url string
	switch c.ContentService.Cfg.Storage.Type {
	case "local":
		dst := filepath.Join(c.ContentService.Cfg.Storage.LocalPath, filename)
		if err := ctx.SaveUploadedFile(file, dst); err != nil {
			util.InternalServerError(ctx)
			return
		}
		url = "/uploads/" + filename
	case "minio":
		minioClient, err := minio.New(c.ContentService.Cfg.Storage.MinioEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(c.ContentService.Cfg.Storage.MinioAccessID, c.ContentService.Cfg.Storage.MinioSecret, ""),
			Secure: false,
		})
		if err != nil {
			util.InternalServerError(ctx)
			return
		}

		src, err := file.Open()
		if err != nil {
			util.InternalServerError(ctx)
			return
		}
		defer src.Close()

		_, err = minioClient.PutObject(ctx, c.ContentService.Cfg.Storage.MinioBucket, filename, src, file.Size, minio.PutObjectOptions{
			ContentType: file.Header.Get("Content-Type"),
		})
		if err != nil {
			util.InternalServerError(ctx)
			return
		}
		url = "/" + c.ContentService.Cfg.Storage.MinioBucket + "/" + filename
	default:
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"success": true,
		"message": "图标上传成功",
		"url":     url,
	})
}

// 视频上传相关内容

// 用于跟踪分块上传进度（不使用Redis的替代方案）
var (
	uploadProgress = make(map[string]*UploadProgress)
	progressMutex  sync.Mutex
)

// UploadProgress 跟踪文件上传进度
type UploadProgress struct {
	TotalChunks    int
	UploadedChunks int
	FileSize       int64
	Identifier     string
	Filename       string
	CreatedAt      time.Time
	Chunks         map[int]bool
}

// VideoUploadRequest 视频上传请求参数
type VideoUploadRequest struct {
	Title       string `form:"title"`
	Description string `form:"description"`
}

// VideoChunkUploadRequest 视频分块上传请求参数
type VideoChunkUploadRequest struct {
	ChunkNumber int    `form:"chunkNumber" binding:"required,min=1"`
	TotalChunks int    `form:"totalChunks" binding:"required,min=1"`
	Identifier  string `form:"identifier" binding:"required,max=100"`
	Filename    string `form:"filename" binding:"required,max=255"`
}

// UploadVideo godoc
// @Summary 上传视频文件（仅管理员）
// @Description 专门用于上传视频文件
// @Tags 内容
// @Accept  multipart/form-data
// @Produce  json
// @Security ApiKeyAuth
// @Param   video formData file true "视频文件"
// @Param   title formData string false "视频标题"
// @Param   description formData string false "视频描述"
// @Success 200 {object} util.Response{data=object} "上传成功"
// @Failure 400 {object} util.Response "请求参数错误或文件格式不支持"
// @Failure 401 {object} util.Response "未授权"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/admin/upload/video [post]
func (c *ContentController) UploadVideo(ctx *gin.Context) {
	var req VideoUploadRequest
	if err := ctx.ShouldBind(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	file, err := ctx.FormFile("video")
	if err != nil {
		util.BadRequest(ctx, "视频文件是必需的")
		return
	}

	// 验证文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validVideoExts := []string{".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv", ".webm"}
	isValidType := false
	for _, e := range validVideoExts {
		if ext == e {
			isValidType = true
			break
		}
	}
	if !isValidType {
		util.BadRequest(ctx, "文件格式不支持，请上传有效的视频文件")
		return
	}

	// 使用当前时间生成唯一文件名
	filename := "videos/" + time.Now().Format("20060102150405") + "-" +
		strings.ReplaceAll(file.Filename, " ", "-")

	// 根据存储类型进行上传
	var url string
	var thumbnailURL string
	var videoPath string // 本地视频文件路径，用于FFmpeg处理

	switch c.ContentService.Cfg.Storage.Type {
	case "local":
		dst := filepath.Join(c.ContentService.Cfg.Storage.LocalPath, filename)
		if err := ctx.SaveUploadedFile(file, dst); err != nil {
			util.InternalServerError(ctx)
			return
		}
		url = "/uploads/" + filename
		videoPath = dst

	case "minio":
		// 对于MinIO存储，我们需要先保存到本地临时文件进行FFmpeg处理
		tempDir := filepath.Join(c.ContentService.Cfg.Storage.LocalPath, "temp")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			util.InternalServerError(ctx)
			return
		}

		// 创建临时文件路径
		tempFilename := fmt.Sprintf("temp_video_%d%s", time.Now().UnixNano(), ext)
		videoPath = filepath.Join(tempDir, tempFilename)

		// 保存临时文件
		if err := ctx.SaveUploadedFile(file, videoPath); err != nil {
			util.InternalServerError(ctx)
			return
		}

		// 上传到MinIO
		minioClient, err := minio.New(c.ContentService.Cfg.Storage.MinioEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(c.ContentService.Cfg.Storage.MinioAccessID, c.ContentService.Cfg.Storage.MinioSecret, ""),
			Secure: false,
		})
		if err != nil {
			util.InternalServerError(ctx)
			return
		}

		src, err := os.Open(videoPath)
		if err != nil {
			util.InternalServerError(ctx)
			return
		}
		defer src.Close()

		_, err = minioClient.PutObject(ctx, c.ContentService.Cfg.Storage.MinioBucket, filename, src, file.Size, minio.PutObjectOptions{
			ContentType: file.Header.Get("Content-Type"),
		})
		if err != nil {
			util.InternalServerError(ctx)
			return
		}
		url = "/" + c.ContentService.Cfg.Storage.MinioBucket + "/" + filename
	default:
		util.InternalServerError(ctx)
		return
	}

	// 生成缩略图
	thumbnailExt := ".jpg"
	thumbnailFilename := "thumbnails/" + time.Now().Format("20060102150405") + "-" +
		strings.ReplaceAll(strings.TrimSuffix(file.Filename, ext), " ", "-") + thumbnailExt

	var thumbnailPath string
	switch c.ContentService.Cfg.Storage.Type {
	case "local":
		// 确保缩略图目录存在
		thumbnailDir := filepath.Join(c.ContentService.Cfg.Storage.LocalPath, "thumbnails")
		if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
			util.InternalServerError(ctx)
			return
		}

		thumbnailPath = filepath.Join(thumbnailDir, filepath.Base(thumbnailFilename))
		thumbnailURL = "/uploads/" + thumbnailFilename

	case "minio":
		// 对于MinIO存储，缩略图先保存到本地
		thumbnailDir := filepath.Join(c.ContentService.Cfg.Storage.LocalPath, "thumbnails")
		if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
			util.InternalServerError(ctx)
			return
		}

		thumbnailPath = filepath.Join(thumbnailDir, filepath.Base(thumbnailFilename))
		thumbnailURL = "/" + c.ContentService.Cfg.Storage.MinioBucket + "/" + thumbnailFilename
	}

	// 使用FFmpeg生成缩略图（从视频3秒的位置抓取一帧）
	err = util.GenerateThumbnail(videoPath, thumbnailPath, "3")
	if err != nil {
		fmt.Printf("生成缩略图失败: %v, 视频路径: %s, 缩略图路径: %s\n", err, videoPath, thumbnailPath)
		// 如果缩略图生成失败，使用默认缩略图
		thumbnailURL = "/uploads/thumbnails/default-video-thumbnail.jpg"
	} else if c.ContentService.Cfg.Storage.Type == "minio" {
		// 如果使用MinIO，上传生成的缩略图
		minioClient, _ := minio.New(c.ContentService.Cfg.Storage.MinioEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(c.ContentService.Cfg.Storage.MinioAccessID, c.ContentService.Cfg.Storage.MinioSecret, ""),
			Secure: false,
		})

		thumbnailFile, err := os.Open(thumbnailPath)
		if err != nil {
			// 上传失败，使用默认缩略图
			thumbnailURL = "/" + c.ContentService.Cfg.Storage.MinioBucket + "/thumbnails/default-video-thumbnail.jpg"
		} else {
			defer thumbnailFile.Close()

			thumbnailInfo, _ := thumbnailFile.Stat()
			minioClient.PutObject(ctx, c.ContentService.Cfg.Storage.MinioBucket, thumbnailFilename,
				thumbnailFile, thumbnailInfo.Size(), minio.PutObjectOptions{ContentType: "image/jpeg"})
		}
	}

	// 使用FFmpeg获取视频时长
	videoInfo, err := util.GetVideoInfo(videoPath)
	var duration float64 = 0
	if err == nil {
		duration = videoInfo.Duration
	}

	// 如果是临时文件，清理
	if c.ContentService.Cfg.Storage.Type == "minio" {
		go func() {
			time.Sleep(5 * time.Second) // 延迟清理，确保FFmpeg处理完成
			os.Remove(videoPath)
		}()
	}

	// 创建资源记录
	resource := &model.Resource{
		Title:       req.Title,
		Description: req.Description,
		Type:        model.Video,
		URL:         url,
		Duration:    duration,
		Size:        file.Size,
		Format:      strings.TrimPrefix(ext, "."),
		Thumbnail:   thumbnailURL,
	}

	if err := c.ContentService.ResourceRepo.Create(resource); err != nil {
		util.InternalServerError(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"id":          resource.ID,
		"url":         url,
		"title":       req.Title,
		"description": req.Description,
		"duration":    duration,
		"size":        file.Size,
		"format":      strings.TrimPrefix(ext, "."),
		"thumbnail":   thumbnailURL,
	})
}

// UploadVideoChunk godoc
// @Summary 上传视频文件分块（仅管理员）
// @Description 支持大视频文件的分块上传
// @Tags 内容
// @Accept  multipart/form-data
// @Produce  json
// @Security ApiKeyAuth
// @Param   chunk formData file true "文件分块数据"
// @Param   chunkNumber formData int true "当前块序号"
// @Param   totalChunks formData int true "总块数"
// @Param   identifier formData string true "文件唯一标识符"
// @Param   filename formData string true "原始文件名"
// @Success 200 {object} util.Response{data=object} "上传成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/admin/upload/video/chunk [post]
func (c *ContentController) UploadVideoChunk(ctx *gin.Context) {
	var req VideoChunkUploadRequest
	if err := ctx.ShouldBind(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	chunkFile, err := ctx.FormFile("chunk")
	if err != nil {
		util.BadRequest(ctx, "分块文件是必需的")
		return
	}

	// 检查块序号是否有效
	if req.ChunkNumber > req.TotalChunks {
		util.BadRequest(ctx, "块序号超过总块数")
		return
	}

	// 创建临时目录存储分块
	tempDir := filepath.Join(c.ContentService.Cfg.Storage.LocalPath, "temp", req.Identifier)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		util.InternalServerError(ctx)
		return
	}

	// 保存分块文件
	chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", req.ChunkNumber))
	if err := ctx.SaveUploadedFile(chunkFile, chunkPath); err != nil {
		util.InternalServerError(ctx)
		return
	}

	// 更新上传进度
	progressMutex.Lock()
	defer progressMutex.Unlock()

	progress, exists := uploadProgress[req.Identifier]
	if !exists {
		progress = &UploadProgress{
			TotalChunks:    req.TotalChunks,
			UploadedChunks: 1,
			FileSize:       chunkFile.Size,
			Identifier:     req.Identifier,
			Filename:       req.Filename,
			CreatedAt:      time.Now(),
			Chunks:         make(map[int]bool),
		}
		uploadProgress[req.Identifier] = progress
	} else {
		progress.FileSize += chunkFile.Size
		if !progress.Chunks[req.ChunkNumber] {
			progress.UploadedChunks++
		}
	}
	progress.Chunks[req.ChunkNumber] = true

	// 检查是否所有分块都已上传
	isComplete := progress.UploadedChunks == progress.TotalChunks

	// 如果所有分块上传完成，合并文件
	var finalURL string
	if isComplete {
		// 创建最终视频文件路径
		ext := filepath.Ext(req.Filename)
		filename := "videos/" + time.Now().Format("20060102150405") + "-" +
			strings.ReplaceAll(strings.TrimSuffix(req.Filename, ext), " ", "-") + ext
		finalPath := filepath.Join(c.ContentService.Cfg.Storage.LocalPath, filename)
		finalURL = "/uploads/" + filename

		// 创建最终文件
		finalFile, err := os.Create(finalPath)
		if err != nil {
			util.InternalServerError(ctx)
			return
		}
		defer finalFile.Close()

		// 按顺序合并所有分块
		for i := 1; i <= req.TotalChunks; i++ {
			chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
			chunkData, err := os.ReadFile(chunkPath)
			if err != nil {
				util.InternalServerError(ctx)
				return
			}

			if _, err := finalFile.Write(chunkData); err != nil {
				util.InternalServerError(ctx)
				return
			}
		}

		// 如果使用MinIO存储，上传合并后的文件
		if c.ContentService.Cfg.Storage.Type == "minio" {
			minioClient, err := minio.New(c.ContentService.Cfg.Storage.MinioEndpoint, &minio.Options{
				Creds:  credentials.NewStaticV4(c.ContentService.Cfg.Storage.MinioAccessID, c.ContentService.Cfg.Storage.MinioSecret, ""),
				Secure: false,
			})
			if err != nil {
				util.InternalServerError(ctx)
				return
			}

			file, err := os.Open(finalPath)
			if err != nil {
				util.InternalServerError(ctx)
				return
			}
			defer file.Close()

			fileInfo, err := file.Stat()
			if err != nil {
				util.InternalServerError(ctx)
				return
			}

			_, err = minioClient.PutObject(ctx, c.ContentService.Cfg.Storage.MinioBucket, filename, file, fileInfo.Size(), minio.PutObjectOptions{
				ContentType: "video/" + strings.TrimPrefix(ext, "."),
			})
			if err != nil {
				util.InternalServerError(ctx)
				return
			}

			finalURL = "/" + c.ContentService.Cfg.Storage.MinioBucket + "/" + filename
		}

		// 清理临时目录和进度记录
		go func() {
			// 延迟清理，确保合并操作完成
			time.Sleep(5 * time.Second)
			os.RemoveAll(tempDir)
			progressMutex.Lock()
			delete(uploadProgress, req.Identifier)
			progressMutex.Unlock()
		}()
	}

	util.Success(ctx, gin.H{
		"identifier":     req.Identifier,
		"chunkNumber":    req.ChunkNumber,
		"totalChunks":    req.TotalChunks,
		"uploadedChunks": progress.UploadedChunks,
		"isComplete":     isComplete,
		"progress":       float64(progress.UploadedChunks) / float64(progress.TotalChunks) * 100,
		"finalURL":       finalURL,
	})
}

// GetUploadProgress godoc
// @Summary 查询视频上传进度（仅管理员）
// @Description 查询文件上传进度
// @Tags 内容
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Param   uploadId path string true "上传标识符"
// @Success 200 {object} util.Response{data=object} "查询成功"
// @Failure 400 {object} util.Response "请求参数错误"
// @Failure 401 {object} util.Response "未授权"
// @Failure 403 {object} util.Response "权限不足"
// @Failure 404 {object} util.Response "上传记录不存在"
// @Failure 500 {object} util.Response "服务器内部错误"
// @Router /api/admin/upload/video/progress/{uploadId} [get]
func (c *ContentController) GetUploadProgress(ctx *gin.Context) {
	uploadId := ctx.Param("uploadId")
	if uploadId == "" {
		util.BadRequest(ctx, "上传标识符不能为空")
		return
	}

	progressMutex.Lock()
	defer progressMutex.Unlock()

	progress, exists := uploadProgress[uploadId]
	if !exists {
		util.NotFound(ctx)
		return
	}

	util.Success(ctx, gin.H{
		"identifier":     progress.Identifier,
		"filename":       progress.Filename,
		"totalChunks":    progress.TotalChunks,
		"uploadedChunks": progress.UploadedChunks,
		"isComplete":     progress.UploadedChunks == progress.TotalChunks,
		"progress":       float64(progress.UploadedChunks) / float64(progress.TotalChunks) * 100,
		"createdAt":      progress.CreatedAt,
	})
}
