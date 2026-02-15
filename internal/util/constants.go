package util

const (
	DateFormat = "2006-01-02"
	TimeFormat = "2006-01-02 15:04:05"
)

const (
	StorageLocal = "local"
	StorageMinio = "minio"
	StorageOSS   = "oss"
)

// 文件上传相关常量
const (
	MimeVideo       = "video/"
	MimeImage       = "image/"
	MimePDF         = "application/pdf"
	MimeOctetStream = "application/octet-stream"
)

var (
	AllowedVideoExtensions = []string{".mp4", ".mov", ".avi", ".mkv", ".wmv", ".flv", ".webm"}
)
