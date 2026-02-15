package service

import (
	"coder_edu_backend/internal/config"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// StorageProvider 定义通用存储接口
type StorageProvider interface {
	Upload(ctx context.Context, filename string, reader io.Reader, size int64, contentType string) (string, error)
	UploadFile(ctx context.Context, filename string, localPath string, contentType string) (string, error)
	Delete(ctx context.Context, filename string) error
	GetURL(filename string) string
}

// LocalStorageProvider 本地存储实现
type LocalStorageProvider struct {
	Config *config.StorageConfig
}

func (p *LocalStorageProvider) Upload(ctx context.Context, filename string, reader io.Reader, size int64, contentType string) (string, error) {
	dst := filepath.Join(p.Config.LocalPath, filename)
	dir := filepath.Dir(dst)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
	}

	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, reader)
	if err != nil {
		return "", err
	}

	return p.GetURL(filename), nil
}

func (p *LocalStorageProvider) UploadFile(ctx context.Context, filename string, localPath string, contentType string) (string, error) {
	dst := filepath.Join(p.Config.LocalPath, filename)
	dir := filepath.Dir(dst)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
	}

	// 如果源文件和目标文件一样，直接返回
	if localPath == dst {
		return p.GetURL(filename), nil
	}

	// 复制文件
	srcFile, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return "", err
	}

	return p.GetURL(filename), nil
}

func (p *LocalStorageProvider) Delete(ctx context.Context, filename string) error {
	dst := filepath.Join(p.Config.LocalPath, filename)
	return os.Remove(dst)
}

func (p *LocalStorageProvider) GetURL(filename string) string {
	return "/uploads/" + filename
}

// MinioStorageProvider MinIO存储实现
type MinioStorageProvider struct {
	Config *config.StorageConfig
	Client *minio.Client
}

func NewMinioStorageProvider(cfg *config.StorageConfig) (*MinioStorageProvider, error) {
	client, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessID, cfg.MinioSecret, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}
	return &MinioStorageProvider{Config: cfg, Client: client}, nil
}

func (p *MinioStorageProvider) Upload(ctx context.Context, filename string, reader io.Reader, size int64, contentType string) (string, error) {
	_, err := p.Client.PutObject(ctx, p.Config.MinioBucket, filename, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return p.GetURL(filename), nil
}

func (p *MinioStorageProvider) UploadFile(ctx context.Context, filename string, localPath string, contentType string) (string, error) {
	_, err := p.Client.FPutObject(ctx, p.Config.MinioBucket, filename, localPath, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return p.GetURL(filename), nil
}

func (p *MinioStorageProvider) Delete(ctx context.Context, filename string) error {
	return p.Client.RemoveObject(ctx, p.Config.MinioBucket, filename, minio.RemoveObjectOptions{})
}

func (p *MinioStorageProvider) GetURL(filename string) string {
	return "/" + p.Config.MinioBucket + "/" + filename
}

// OSSStorageProvider 阿里云OSS存储实现
type OSSStorageProvider struct {
	Config *config.StorageConfig
	Client *oss.Client
}

func NewOSSStorageProvider(cfg *config.StorageConfig) (*OSSStorageProvider, error) {
	client, err := oss.New(cfg.OSSEndpoint, cfg.OSSAccessKey, cfg.OSSSecretKey)
	if err != nil {
		return nil, err
	}
	return &OSSStorageProvider{Config: cfg, Client: client}, nil
}

func (p *OSSStorageProvider) Upload(ctx context.Context, filename string, reader io.Reader, size int64, contentType string) (string, error) {
	bucket, err := p.Client.Bucket(p.Config.OSSBucket)
	if err != nil {
		return "", err
	}

	err = bucket.PutObject(filename, reader)
	if err != nil {
		return "", err
	}
	return p.GetURL(filename), nil
}

func (p *OSSStorageProvider) UploadFile(ctx context.Context, filename string, localPath string, contentType string) (string, error) {
	bucket, err := p.Client.Bucket(p.Config.OSSBucket)
	if err != nil {
		return "", err
	}

	err = bucket.PutObjectFromFile(filename, localPath)
	if err != nil {
		return "", err
	}
	return p.GetURL(filename), nil
}

func (p *OSSStorageProvider) Delete(ctx context.Context, filename string) error {
	bucket, err := p.Client.Bucket(p.Config.OSSBucket)
	if err != nil {
		return err
	}
	return bucket.DeleteObject(filename)
}

func (p *OSSStorageProvider) GetURL(filename string) string {
	return fmt.Sprintf("https://%s.%s/%s", p.Config.OSSBucket, p.Config.OSSEndpoint, filename)
}

// StorageService 存储服务
type StorageService struct {
	Provider StorageProvider
}

func NewStorageService(cfg *config.Config) *StorageService {
	var provider StorageProvider
	switch cfg.Storage.Type {
	case "minio":
		p, err := NewMinioStorageProvider(&cfg.Storage)
		if err == nil {
			provider = p
		}
	case "oss":
		p, err := NewOSSStorageProvider(&cfg.Storage)
		if err == nil {
			provider = p
		}
	}

	if provider == nil {
		provider = &LocalStorageProvider{Config: &cfg.Storage}
	}

	return &StorageService{Provider: provider}
}

func (s *StorageService) Upload(ctx context.Context, filename string, reader io.Reader, size int64, contentType string) (string, error) {
	return s.Provider.Upload(ctx, filename, reader, size, contentType)
}

func (s *StorageService) UploadFile(ctx context.Context, filename string, localPath string, contentType string) (string, error) {
	return s.Provider.UploadFile(ctx, filename, localPath, contentType)
}

func (s *StorageService) Delete(ctx context.Context, filename string) error {
	return s.Provider.Delete(ctx, filename)
}

func (s *StorageService) GetURL(filename string) string {
	return s.Provider.GetURL(filename)
}
