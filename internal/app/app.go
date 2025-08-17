package app

import (
	"coder_edu_backend/docs"
	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/controller"
	"coder_edu_backend/internal/middleware"
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/pkg/configwatcher"
	"coder_edu_backend/pkg/database"
	"coder_edu_backend/pkg/logger"
	"coder_edu_backend/pkg/monitoring"
	"coder_edu_backend/pkg/security"
	"coder_edu_backend/pkg/tracing"
	"context"
	"log"
	"path/filepath"
	"time"

	_ "coder_edu_backend/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	Config          *config.Config
	Router          *gin.Engine
	DB              *gorm.DB
	configCallbacks []func(*config.Config)
}

func (a *App) RegisterConfigCallback(callback func(*config.Config)) {
	a.configCallbacks = append(a.configCallbacks, callback)
}
func NewApp(cfg *config.Config) *App {

	logger.InitLogger(cfg)
	defer logger.Log.Sync()

	logger.Log.Info("Logger initialized successfully")

	db, err := database.InitDB(&cfg.Database)
	if err != nil {
		logger.Log.Fatal("Failed to initialize database", zap.Error(err))
		log.Fatalf("Failed to initialize database: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	resourceRepo := repository.NewResourceRepository(db)

	authService := service.NewAuthService(userRepo, cfg)
	contentService := service.NewContentService(resourceRepo, cfg)

	authController := controller.NewAuthController(authService)
	contentController := controller.NewContentController(contentService)

	// 监控
	monitoring.Init()

	router := gin.Default()

	router.Use(security.CORS())
	router.Use(security.Secure())
	router.Use(security.RateLimiter(100, time.Minute)) // 每分钟100次请求

	// 初始化分布式追踪
	if cfg.Tracing.Enabled {
		tp, err := tracing.InitTracer("learning-platform", cfg.Tracing.CollectorEndpoint)
		if err != nil {
			logger.Log.Fatal("Failed to initialize tracing", zap.Error(err))
		}
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				logger.Log.Error("Failed to shutdown tracer provider", zap.Error(err))
			}
		}()

		router.Use(tracing.GinMiddleware())
	}

	docs.SwaggerInfo.BasePath = "/api"
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")))

	router.Use(monitoring.MetricsMiddleware())

	router.Use(func(c *gin.Context) {
		c.Set("config", cfg)
		c.Next()
	})

	router.GET("/metrics", monitoring.PrometheusHandler())

	healthController := controller.NewHealthController(db)

	public := router.Group("/api")
	{
		public.GET("/health", healthController.HealthCheck)
		public.POST("/register", authController.Register)
		public.POST("/login", authController.Login)
	}

	auth := router.Group("/api")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.GET("/profile", authController.GetProfile)
		auth.GET("/resources", contentController.GetResources)
	}

	admin := router.Group("/api/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(model.Admin))
	{
		admin.POST("/resources", contentController.UploadResource)
	}

	if cfg.Storage.Type == "local" {
		router.Static("/uploads", cfg.Storage.LocalPath)
	}

	return &App{
		Config: cfg,
		Router: router,
		DB:     db,
	}
}

func (a *App) Run() {
	log.Printf("Server running on port %s", a.Config.Server.Port)

	// 启动配置热加载
	go func() {
		configDir := "configs"
		configFile := "config.yaml"
		configPath := filepath.Join(configDir, configFile)
		configwatcher.WatchConfig(configPath, a.Config, func(newCfg interface{}) {
			logger.Log.Info("Config reloaded")

			newConfig, ok := newCfg.(*config.Config)
			if !ok {
				logger.Log.Error("Failed to cast new config to Config type")
				return
			}

			a.Config = newConfig
			for _, callback := range a.configCallbacks {
				callback(newConfig)
			}

			logger.InitLogger(a.Config)
			logger.Log.Info("Logger reinitialized with new config")
		})
	}()

	if err := a.Router.Run(":" + a.Config.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
