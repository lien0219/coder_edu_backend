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
	"log"
)

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	cfg, err := config.LoadConfig("configs")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	application := app.NewApp(cfg)
	defer logger.Log.Sync()
	application.Run()
}
