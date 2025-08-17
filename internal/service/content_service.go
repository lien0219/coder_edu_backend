package service

import (
	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"coder_edu_backend/internal/util"
	"errors"
	"mime/multipart"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type ContentService struct {
	ResourceRepo *repository.ResourceRepository
	Cfg          *config.Config
}

func NewContentService(resourceRepo *repository.ResourceRepository, cfg *config.Config) *ContentService {
	return &ContentService{
		ResourceRepo: resourceRepo,
		Cfg:          cfg,
	}
}

func (s *ContentService) UploadResource(c *gin.Context, file *multipart.FileHeader, resource *model.Resource) error {
	claims := util.GetUserFromContext(c)
	if claims == nil {
		return errors.New("unauthorized")
	}

	resource.UploaderID = claims.UserID

	switch s.Cfg.Storage.Type {
	case "local":
		return s.uploadLocal(c, file, resource)
	case "minio":
		return s.uploadMinio(c, file, resource)
	default:
		return errors.New("unsupported storage type")
	}
}

func (s *ContentService) uploadLocal(c *gin.Context, file *multipart.FileHeader, resource *model.Resource) error {
	dst := filepath.Join(s.Cfg.Storage.LocalPath, file.Filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		return err
	}

	resource.URL = "/uploads/" + file.Filename
	return s.ResourceRepo.Create(resource)
}

func (s *ContentService) uploadMinio(c *gin.Context, file *multipart.FileHeader, resource *model.Resource) error {
	minioClient, err := minio.New(s.Cfg.Storage.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s.Cfg.Storage.MinioAccessID, s.Cfg.Storage.MinioSecret, ""),
		Secure: false,
	})
	if err != nil {
		return err
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	ext := filepath.Ext(file.Filename)
	filename := time.Now().Format("20060102150405") + ext

	_, err = minioClient.PutObject(c, s.Cfg.Storage.MinioBucket, filename, src, file.Size, minio.PutObjectOptions{
		ContentType: file.Header.Get("Content-Type"),
	})
	if err != nil {
		return err
	}

	resource.URL = "/" + s.Cfg.Storage.MinioBucket + "/" + filename
	return s.ResourceRepo.Create(resource)
}
