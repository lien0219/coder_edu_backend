// 手动触发 AI 自动打标签脚本
//
// 该功能已集成到主应用的后台定时任务中（每 24 小时自动执行一次）。
// 此脚本仅用于手动触发，例如首次部署或数据库大量导入新数据后。
//
// 用法: go run scripts/auto_tagging.go

package main

import (
	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/pkg/database"
	"coder_edu_backend/pkg/logger"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	data, err := os.ReadFile("configs/config.yaml")
	if err != nil {
		log.Fatalf("无法读取配置文件: %v", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}

	logger.InitLogger(&cfg)

	db, err := database.InitDB(&cfg.Database, cfg.Server.Mode)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	aiService := service.NewAIService(cfg.AI)
	autoTagging := service.NewAutoTaggingService(db, aiService)

	log.Println("手动触发自动打标签任务...")
	autoTagging.RunAutoTagging()
	log.Println("完成！")
}
