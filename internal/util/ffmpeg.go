package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

// VideoInfo 存储视频信息
type VideoInfo struct {
	Duration float64 `json:"duration"` // 视频时长（秒）
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Format   string  `json:"format"`
	Size     int64   `json:"size"`
}

// GetVideoInfo 使用ffmpeg-go库获取视频信息
func GetVideoInfo(videoPath string) (*VideoInfo, error) {
	// 检查文件是否存在
	fileInfo, err := os.Stat(videoPath)
	if err != nil {
		return nil, fmt.Errorf("视频文件不存在: %v", err)
	}

	// 使用ffmpeg-go的Probe函数获取视频元数据
	jsonOutput, err := ffmpeg.Probe(videoPath)
	if err != nil {
		return nil, fmt.Errorf("获取视频信息失败: %v", err)
	}

	// 解析JSON输出
	var result struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			Size     string `json:"size"`
			Format   string `json:"format_name"`
		} `json:"format"`
	}

	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		return nil, fmt.Errorf("解析视频信息失败: %v", err)
	}

	// 提取视频流信息
	var width, height int
	for _, stream := range result.Streams {
		if stream.CodecType == "video" {
			width = stream.Width
			height = stream.Height
			break
		}
	}

	// 解析时长
	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		duration = 0
	}

	// 解析文件大小
	size, err := strconv.ParseInt(result.Format.Size, 10, 64)
	if err != nil {
		size = fileInfo.Size()
	}

	// 提取格式信息
	format := "unknown"
	if len(result.Format.Format) > 0 {
		formatParts := strings.Split(result.Format.Format, ",")
		if len(formatParts) > 0 {
			format = formatParts[0]
		}
	}

	return &VideoInfo{
		Duration: duration,
		Width:    width,
		Height:   height,
		Format:   format,
		Size:     size,
	}, nil
}

// GenerateThumbnail 使用ffmpeg-go库生成视频缩略图
func GenerateThumbnail(videoPath, thumbnailPath string, timeOffset string) error {
	// 确保目录存在
	dir := strings.Replace(thumbnailPath, "\\", "/", -1)
	dirParts := strings.Split(dir, "/")
	if len(dirParts) > 1 {
		dir = strings.Join(dirParts[:len(dirParts)-1], "/")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建缩略图目录失败: %v", err)
	}

	// 使用ffmpeg-go的链式API生成缩略图
	// 使用Run函数直接执行命令，不使用Stderr
	return ffmpeg.Input(videoPath, ffmpeg.KwArgs{
		"ss": timeOffset, // 从视频的哪个时间点抓取帧
	}).
		Output(thumbnailPath, ffmpeg.KwArgs{
			"vframes": "1", // 只抓取一帧
			"q:v":     "2", // 图像质量 (1-31, 越小质量越高)
		}).
		OverWriteOutput().
		Run()
}

// GetFFmpegVersion 获取FFmpeg版本信息，用于检查FFmpeg是否正确安装
func GetFFmpegVersion() (string, error) {
	// 使用标准库os/exec直接调用ffmpeg命令，因为ffmpeg-go库没有NewCommand方法
	cmd := exec.Command("ffmpeg", "-version", "-hide_banner")
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("获取FFmpeg版本失败，请确保FFmpeg已正确安装: %v, %s", err, errOut.String())
	}

	return out.String(), nil
}
