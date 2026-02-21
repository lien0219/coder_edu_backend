package main

import (
	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/pkg/database"
	"fmt"
	"log"
	"os"
	"strings"

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

	db, err := database.InitDB(&cfg.Database, cfg.Server.Mode)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	aiService := service.NewAIService(cfg.AI)

	var kps []model.KnowledgePoint
	db.Find(&kps)

	var exercises []model.ExerciseQuestion
	db.Find(&exercises)

	fmt.Printf("开始为 %d 个知识点和 %d 个练习题自动生成标签...\n", len(kps), len(exercises))

	// 自动打标签逻辑
	for _, kp := range kps {
		prompt := fmt.Sprintf("请为以下编程知识点提取 3-5 个核心关键词标签，仅返回标签，用逗号分隔。\n标题: %s\n内容: %s", kp.Title, kp.ArticleContent)
		tags, err := aiService.Chat(prompt, "")
		if err == nil {
			tags = strings.ReplaceAll(tags, " ", "")
			fmt.Printf("知识点 [%s] -> 标签: %s\n", kp.Title, tags)
		}
	}

	for _, ex := range exercises {
		prompt := fmt.Sprintf("请为以下编程练习题提取 3-5 个核心关键词标签，仅返回标签，用逗号分隔。\n标题: %s\n描述: %s", ex.Title, ex.Description)
		tags, err := aiService.Chat(prompt, "")
		if err == nil {
			tags = strings.ReplaceAll(tags, " ", "")
			fmt.Printf("练习题 [%s] -> 标签: %s\n", ex.Title, tags)
		}
	}

	fmt.Println("自动打标签任务完成！")
}
