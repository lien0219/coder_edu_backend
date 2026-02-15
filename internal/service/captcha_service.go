package service

import (
	"coder_edu_backend/internal/config"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type CaptchaService struct {
	Redis *redis.Client
	Cfg   *config.Config
}

type TrajectoryPoint struct {
	X int `json:"x"`
	Y int `json:"y"`
	T int `json:"t"`
}

func NewCaptchaService(rdb *redis.Client, cfg *config.Config) *CaptchaService {
	return &CaptchaService{
		Redis: rdb,
		Cfg:   cfg,
	}
}

// VerifyTrajectory 校验滑动轨迹
func (s *CaptchaService) VerifyTrajectory(trajectory []TrajectoryPoint, duration int) (string, error) {
	if len(trajectory) < 10 {
		return "", fmt.Errorf("trajectory too short")
	}

	if !s.analyzeTrajectory(trajectory, duration) {
		return "", fmt.Errorf("human machine verification failed")
	}

	captchaToken := uuid.New().String()
	ctx := context.Background()

	// 有效期2分钟
	err := s.Redis.Set(ctx, "captcha:"+captchaToken, "verified", 2*time.Minute).Err()
	if err != nil {
		return "", err
	}

	return captchaToken, nil
}

// ValidateToken 验证 Captcha Token
func (s *CaptchaService) ValidateToken(token string) bool {
	if token == "" {
		return false
	}

	ctx := context.Background()
	key := "captcha:" + token
	val, err := s.Redis.Get(ctx, key).Result()
	if err != nil || val != "verified" {
		return false
	}

	s.Redis.Del(ctx, key)
	return true
}

// GenerateTrustDeviceToken 生成 15 天免验证 Token
func (s *CaptchaService) GenerateTrustDeviceToken(userID uint) (string, error) {
	// 使用时间戳和随机数生成Token，并进行签名或加密
	rawToken := fmt.Sprintf("trust:%d:%d:%s", userID, time.Now().Unix(), uuid.New().String())
	token := generateHash(rawToken)

	ctx := context.Background()
	key := "trust_device:" + token
	// 存入Redis，有效期 15 天
	err := s.Redis.Set(ctx, key, userID, 15*24*time.Hour).Err()
	if err != nil {
		return "", err
	}

	return token, nil
}

// VerifyTrustDeviceToken 验证免验证 Token
func (s *CaptchaService) VerifyTrustDeviceToken(token string) (uint, bool) {
	if token == "" {
		return 0, false
	}

	ctx := context.Background()
	key := "trust_device:" + token
	val, err := s.Redis.Get(ctx, key).Uint64()
	if err != nil {
		return 0, false
	}

	return uint(val), true
}

// analyzeTrajectory 模拟轨迹分析算法
func (s *CaptchaService) analyzeTrajectory(trajectory []TrajectoryPoint, duration int) bool {
	if duration < 200 || duration > 10000 {
		return false
	}

	var totalDistance float64
	var totalJitter float64

	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		dist := math.Sqrt(dx*dx + dy*dy)
		totalDistance += dist

		// 简单的抖动检测
		if math.Abs(dy) > 10 {
			totalJitter += math.Abs(dy)
		}
	}

	if totalJitter == 0 && len(trajectory) < 20 {
		// return false // 暂时放宽，生产中需更严格
	}

	// 50是预估值
	if totalDistance < 50 {
		return false
	}

	return true
}

func generateHash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
