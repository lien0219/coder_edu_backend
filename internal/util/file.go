package util

import (
	"errors"
	"io"
	"net/http"
	"strings"
)

// ValidateMimeType 深度校验文件 MIME 类型
// allowedTypes: 允许的 MIME 前缀或完整类型，如 "image/", "video/", "application/pdf"
func ValidateMimeType(reader io.Reader, allowedTypes []string) (string, error) {
	buffer := make([]byte, 512)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	// 检测 MIME 类型
	mimeType := http.DetectContentType(buffer[:n])

	for _, allowed := range allowedTypes {
		if strings.HasPrefix(mimeType, allowed) || mimeType == allowed {
			return mimeType, nil
		}
	}

	return mimeType, errors.New("invalid file type: " + mimeType)
}

// IsImage 检测是否为图片
func IsImage(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

// IsVideo 检测是否为视频
func IsVideo(mimeType string) bool {
	return strings.HasPrefix(mimeType, "video/") || mimeType == "application/x-mpegURL"
}
