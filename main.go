// @title CoderEdu 后端 API
// @version 1.0
// @description CoderEdu学习平台的后端服务器。
// @termsOfService http://swagger.io/terms/

// @contact.name API支持
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

package main

import (
	"coder_edu_backend/internal/app"
	"coder_edu_backend/internal/config"
	"coder_edu_backend/pkg/logger"
	"flag"
	"log"
)

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	// 命令行参数
	migrateOnly := flag.Bool("migrate-only", false, "只执行数据库迁移，完成后退出")
	migrate := flag.Bool("migrate", false, "启动时强制执行数据库迁移（即使是 release 模式）")
	flag.Parse()

	cfg, err := config.LoadConfig("configs")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 设置迁移标志
	cfg.ForceMigrate = *migrate || *migrateOnly
	cfg.MigrateOnly = *migrateOnly

	application := app.NewApp(cfg)
	defer logger.Log.Sync()

	// 迁移完成后直接退出
	if *migrateOnly {
		log.Println("数据库迁移完成，退出程序")
		return
	}

	application.Run()
}
