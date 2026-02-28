package service

import (
	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/util"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// StorageProvider 定义通用存储接口
type StorageProvider interface {
	Upload(ctx context.Context, filename string, reader io.Reader, size int64, contentType string) (string, error)
	UploadFile(ctx context.Context, filename string, localPath string, contentType string) (string, error)
	UploadChunks(ctx context.Context, filename string, chunkPaths []string, contentType string) (string, error)
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

func (p *LocalStorageProvider) UploadChunks(ctx context.Context, filename string, chunkPaths []string, contentType string) (string, error) {
	finalPath := filepath.Join(p.Config.LocalPath, "temp", fmt.Sprintf("merged_%d_%s", time.Now().UnixNano(), filepath.Base(filename)))
	defer os.Remove(finalPath)

	// 合并分块
	finalFile, err := os.Create(finalPath)
	if err != nil {
		return "", err
	}
	defer finalFile.Close()

	buf := make([]byte, 32*1024) // 32KB 缓冲区
	for _, chunkPath := range chunkPaths {
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return "", err
		}
		_, err = io.CopyBuffer(finalFile, chunkFile, buf)
		chunkFile.Close()
		if err != nil {
			return "", err
		}
	}
	finalFile.Close()

	// 上传合并后的文件
	return p.UploadFile(ctx, filename, finalPath, contentType)
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

func (p *MinioStorageProvider) UploadChunks(ctx context.Context, filename string, chunkPaths []string, contentType string) (string, error) {
	// MinIO也先合并再上传（后续优化为MinIO分片上传）
	finalPath := filepath.Join(p.Config.LocalPath, "temp", fmt.Sprintf("merged_%d_%s", time.Now().UnixNano(), filepath.Base(filename)))
	defer os.Remove(finalPath)

	finalFile, err := os.Create(finalPath)
	if err != nil {
		return "", err
	}
	defer finalFile.Close()

	buf := make([]byte, 32*1024)
	for _, chunkPath := range chunkPaths {
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return "", err
		}
		_, err = io.CopyBuffer(finalFile, chunkFile, buf)
		chunkFile.Close()
		if err != nil {
			return "", err
		}
	}
	finalFile.Close()

	return p.UploadFile(ctx, filename, finalPath, contentType)
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

	options := []oss.Option{
		oss.ContentType(contentType),
		oss.ContentDisposition("inline"),
	}

	err = bucket.PutObject(filename, reader, options...)
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

	// 获取文件大小
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return "", err
	}
	fileSize := fileInfo.Size()

	// 所有文件都使用分片上传，提高上传速度和可靠性
	return p.uploadFileMultipart(ctx, bucket, filename, localPath, contentType, fileSize)
}

// uploadFileMultipart 使用分片上传文件
func (p *OSSStorageProvider) uploadFileMultipart(ctx context.Context, bucket *oss.Bucket, filename, localPath, contentType string, fileSize int64) (string, error) {
	// 分片大小：10MB
	chunkSize := int64(10 * 1024 * 1024)

	// 对于小于分片大小的文件，直接使用一个分片
	if fileSize <= chunkSize {
		chunkSize = fileSize
	}

	chunkCount := int((fileSize + chunkSize - 1) / chunkSize)

	// 初始化分片上传
	imur, err := bucket.InitiateMultipartUpload(filename, oss.ContentType(contentType), oss.ContentDisposition("inline"))
	if err != nil {
		return "", fmt.Errorf("初始化分片上传失败: %w", err)
	}

	// 打开文件
	file, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 并发上传分片（最多5个并发）
	type partResult struct {
		partNumber int
		part       oss.UploadPart
		err        error
	}
	semaphore := make(chan struct{}, 5)
	results := make(chan partResult, chunkCount)

	// 上传所有分片
	for i := 0; i < chunkCount; i++ {
		go func(partNumber int) {
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			offset := int64(partNumber) * chunkSize
			length := chunkSize
			if offset+length > fileSize {
				length = fileSize - offset
			}

			// 读取分片数据
			reader := io.NewSectionReader(file, offset, length)
			part, err := bucket.UploadPart(imur, reader, length, partNumber+1, oss.ContentType(contentType))
			results <- partResult{partNumber: partNumber, part: part, err: err}
		}(i)
	}

	// 收集所有分片结果并按partNumber排序
	partMap := make(map[int]oss.UploadPart, chunkCount)
	for i := 0; i < chunkCount; i++ {
		result := <-results
		if result.err != nil {
			bucket.AbortMultipartUpload(imur)
			return "", fmt.Errorf("上传分片 %d 失败: %w", result.partNumber+1, result.err)
		}
		partMap[result.partNumber] = result.part
	}

	// 按partNumber顺序构建parts数组
	parts := make([]oss.UploadPart, chunkCount)
	for i := 0; i < chunkCount; i++ {
		parts[i] = partMap[i]
	}

	// 完成分片上传
	_, err = bucket.CompleteMultipartUpload(imur, parts)
	if err != nil {
		bucket.AbortMultipartUpload(imur)
		return "", fmt.Errorf("完成分片上传失败: %w", err)
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
	publicEndpoint := strings.Replace(p.Config.OSSEndpoint, "-internal", "", 1)
	return fmt.Sprintf("https://%s.%s/%s", p.Config.OSSBucket, publicEndpoint, filename)
}

// UploadChunks 直接从分块上传到OSS（跳过本地合并步骤）
func (p *OSSStorageProvider) UploadChunks(ctx context.Context, filename string, chunkPaths []string, contentType string) (string, error) {
	bucket, err := p.Client.Bucket(p.Config.OSSBucket)
	if err != nil {
		return "", err
	}

	chunkCount := len(chunkPaths)
	if chunkCount == 0 {
		return "", fmt.Errorf("分块列表为空")
	}

	// 初始化分片上传
	imur, err := bucket.InitiateMultipartUpload(filename, oss.ContentType(contentType), oss.ContentDisposition("inline"))
	if err != nil {
		return "", fmt.Errorf("初始化分片上传失败: %w", err)
	}

	// 并发上传分片（最多5个并发）
	type partResult struct {
		partNumber int
		part       oss.UploadPart
		err        error
	}
	semaphore := make(chan struct{}, 5)
	results := make(chan partResult, chunkCount)

	// 上传所有分片
	for i, chunkPath := range chunkPaths {
		go func(partNumber int, chunkPath string) {
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			// 打开分块文件
			chunkFile, err := os.Open(chunkPath)
			if err != nil {
				results <- partResult{partNumber: partNumber, err: fmt.Errorf("打开分块文件失败: %w", err)}
				return
			}
			defer chunkFile.Close()

			// 获取文件大小
			fileInfo, err := chunkFile.Stat()
			if err != nil {
				results <- partResult{partNumber: partNumber, err: fmt.Errorf("获取分块文件信息失败: %w", err)}
				return
			}
			fileSize := fileInfo.Size()

			// 上传分片
			part, err := bucket.UploadPart(imur, chunkFile, fileSize, partNumber+1, oss.ContentType(contentType))
			if err != nil {
				results <- partResult{partNumber: partNumber, err: fmt.Errorf("上传分片失败: %w", err)}
				return
			}

			results <- partResult{partNumber: partNumber, part: part, err: nil}
		}(i, chunkPath)
	}

	// 收集所有分片结果并按partNumber排序
	partMap := make(map[int]oss.UploadPart, chunkCount)
	for i := 0; i < chunkCount; i++ {
		result := <-results
		if result.err != nil {
			bucket.AbortMultipartUpload(imur)
			return "", fmt.Errorf("上传分片 %d 失败: %w", result.partNumber+1, result.err)
		}
		partMap[result.partNumber] = result.part
	}

	// 按partNumber顺序构建parts数组
	parts := make([]oss.UploadPart, chunkCount)
	for i := 0; i < chunkCount; i++ {
		parts[i] = partMap[i]
	}

	// 完成分片上传
	_, err = bucket.CompleteMultipartUpload(imur, parts)
	if err != nil {
		bucket.AbortMultipartUpload(imur)
		return "", fmt.Errorf("完成分片上传失败: %w", err)
	}

	return p.GetURL(filename), nil
}

// StorageService 存储服务
type StorageService struct {
	Provider StorageProvider
}

func NewStorageService(cfg *config.Config) *StorageService {
	var provider StorageProvider
	switch cfg.Storage.Type {
	case util.StorageMinio:
		p, err := NewMinioStorageProvider(&cfg.Storage)
		if err == nil {
			provider = p
		}
	case util.StorageOSS:
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

func (s *StorageService) UploadChunks(ctx context.Context, filename string, chunkPaths []string, contentType string) (string, error) {
	return s.Provider.UploadChunks(ctx, filename, chunkPaths, contentType)
}
