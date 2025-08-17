// @title CoderEdu Backend API
// @version 1.0
// @description This is a backend server for CoderEdu learning platform.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
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
	application.Run()
}
