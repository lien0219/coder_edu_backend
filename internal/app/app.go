package app

import (
	"coder_edu_backend/docs"
	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/controller"
	"coder_edu_backend/internal/middleware"
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/pkg/database"
	"coder_edu_backend/pkg/logger"
	"coder_edu_backend/pkg/monitoring"
	"coder_edu_backend/pkg/security"
	"coder_edu_backend/pkg/tracing"
	"context"
	"log"
	"strconv"
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
	taskRepo := repository.NewTaskRepository(db)
	goalRepo := repository.NewGoalRepository(db)
	moduleRepo := repository.NewModuleRepository(db)
	progressRepo := repository.NewProgressRepository(db)
	learningLogRepo := repository.NewLearningLogRepository(db)
	quizRepo := repository.NewQuizRepository(db)
	achievementRepo := repository.NewAchievementRepository(db)
	postRepo := repository.NewPostRepository(db)
	questionRepo := repository.NewQuestionRepository(db)
	answerRepo := repository.NewAnswerRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	skillRepo := repository.NewSkillRepository(db)
	recommendationRepo := repository.NewRecommendationRepository(db)
	motivationRepo := repository.NewMotivationRepository(db)
	cProgrammingResRepo := repository.NewCProgrammingResourceRepository(db)
	exerciseCategoryRepo := repository.NewExerciseCategoryRepository(db)
	exerciseQuestionRepo := repository.NewExerciseQuestionRepository(db)
	exerciseSubmissionRepo := repository.NewExerciseSubmissionRepository(db)

	authService := service.NewAuthService(userRepo, cfg)
	contentService := service.NewContentService(resourceRepo, cfg)
	motivationService := service.NewMotivationService(motivationRepo)
	dashboardService := service.NewDashboardService(userRepo, taskRepo, resourceRepo, goalRepo, motivationService)
	learningService := service.NewLearningService(moduleRepo, taskRepo, resourceRepo, progressRepo, learningLogRepo, quizRepo, db)
	achievementService := service.NewAchievementService(achievementRepo, userRepo, goalRepo)
	communityService := service.NewCommunityService(postRepo, nil, questionRepo, answerRepo, userRepo)
	analyticsService := service.NewAnalyticsService(progressRepo, sessionRepo, skillRepo, learningLogRepo, recommendationRepo)
	userService := service.NewUserService(userRepo)
	cProgrammingResService := service.NewCProgrammingResourceService(
		cProgrammingResRepo,
		exerciseCategoryRepo,
		exerciseQuestionRepo,
		exerciseSubmissionRepo,
		resourceRepo,
		db,
	)

	authController := controller.NewAuthController(authService)
	contentController := controller.NewContentController(contentService)
	motivationController := controller.NewMotivationController(motivationService)
	dashboardController := controller.NewDashboardController(dashboardService)
	learningController := controller.NewLearningController(learningService)
	achievementController := controller.NewAchievementController(achievementService)
	communityController := controller.NewCommunityController(communityService)
	analyticsController := controller.NewAnalyticsController(analyticsService)
	userController := controller.NewUserController(userService)
	cProgrammingResController := controller.NewCProgrammingResourceController(cProgrammingResService, contentService)

	// 监控
	monitoring.Init()

	router := gin.Default()

	router.Use(security.CORS())
	router.Use(security.Secure())
	router.Use(security.RateLimiter(300, time.Minute)) // 每分钟300次请求

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
		public.GET("/motivation", motivationController.GetCurrentMotivation)
		public.POST("/users/:id/points", userController.UpdateUserPoints)
	}

	// 用于无需权限的答案提交接口
	publicAPI := router.Group("/api/public")
	{
		publicAPI.POST("/c-programming/questions/:questionId/submit", cProgrammingResController.SubmitExerciseAnswerPublic)
	}

	auth := router.Group("/api")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.GET("/profile", authController.GetProfile)
		auth.GET("/resources", contentController.GetResources)

		auth.GET("/dashboard", dashboardController.GetDashboard)
		auth.GET("/dashboard/today-tasks", dashboardController.GetTodayTasks)
		auth.PATCH("/dashboard/tasks/:taskId", dashboardController.UpdateTaskStatus)

		auth.GET("/learning/pre-class", learningController.GetPreClass)
		auth.GET("/learning/in-class", learningController.GetInClass)
		auth.GET("/learning/post-class", learningController.GetPostClass)
		auth.POST("/learning/learning-log", learningController.SubmitLearningLog)
		auth.POST("/learning/quiz/:quizId", learningController.SubmitQuiz)

		auth.GET("/achievements", achievementController.GetUserAchievements)
		auth.GET("/achievements/leaderboard", achievementController.GetLeaderboard)
		auth.GET("/achievements/goals", achievementController.GetUserGoals)
		auth.POST("/achievements/goals", achievementController.CreateGoal)
		auth.PATCH("/achievements/goals/:goalId", achievementController.UpdateGoalProgress)

		auth.GET("/community/posts", communityController.GetPosts)
		auth.POST("/community/posts", communityController.CreatePost)
		auth.GET("/community/questions", communityController.GetQuestions)
		auth.POST("/community/questions", communityController.CreateQuestion)
		auth.POST("/community/questions/:questionId/answers", communityController.AnswerQuestion)
		auth.POST("/community/:type/:id/upvote", communityController.Upvote)

		auth.GET("/analytics/overview", analyticsController.GetOverview)
		auth.GET("/analytics/progress", analyticsController.GetProgress)
		auth.GET("/analytics/skills", analyticsController.GetSkills)
		auth.GET("/analytics/recommendations", analyticsController.GetRecommendations)
		auth.POST("/analytics/session/start", analyticsController.StartSession)
		auth.POST("/analytics/session/:sessionId/end", analyticsController.EndSession)

		auth.GET("/c-programming/resources", cProgrammingResController.GetResources)
		auth.GET("/c-programming/resources/full", cProgrammingResController.GetResourcesWithAllContent)
		auth.GET("/c-programming/resources/:id", cProgrammingResController.GetResourceByID)
		auth.GET("/c-programming/resources/:id/categories", cProgrammingResController.GetCategoriesByResourceID)
		auth.GET("/c-programming/categories/:categoryId/questions", cProgrammingResController.GetQuestionsByCategoryID)
		auth.GET("/c-programming/categories/:categoryId/questions-with-status", cProgrammingResController.GetQuestionsByCategoryIDWithUserStatus)
		auth.GET("/c-programming/resources/:id/videos", cProgrammingResController.GetVideosByResourceID)
		auth.GET("/c-programming/resources/:id/articles", cProgrammingResController.GetArticlesByResourceID)

		auth.GET("/c-programming/exercises/users/:userID/questions/:questionID/submission", cProgrammingResController.CheckUserSubmittedQuestion)
	}

	admin := router.Group("/api/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(model.Admin))
	{
		admin.POST("/upload/icon", contentController.UploadIcon)
		admin.POST("/resources", contentController.UploadResource)
		admin.GET("/users", userController.GetUsers)
		admin.GET("/users/:id", userController.GetUser)
		admin.PUT("/users/:id", userController.UpdateUser)
		admin.DELETE("/users/:id", userController.DeleteUser)
		admin.POST("/users/:id/reset-password", userController.ResetPassword)
		admin.POST("/users/:id/disable", userController.DisableUser)

		admin.GET("/motivations", motivationController.GetAllMotivations)
		admin.POST("/motivations", motivationController.CreateMotivation)
		admin.PUT("/motivations/:id", motivationController.UpdateMotivation)
		admin.DELETE("/motivations/:id", motivationController.DeleteMotivation)
		admin.POST("/motivations/:id/switch", motivationController.SwitchMotivation)

		admin.POST("/c-programming/resources", cProgrammingResController.CreateResource)
		admin.PUT("/c-programming/resources/:id", cProgrammingResController.UpdateResource)
		admin.DELETE("/c-programming/resources/:id", cProgrammingResController.DeleteResource)
		admin.POST("/c-programming/resources/:id/categories", cProgrammingResController.CreateCategory)
		admin.POST("/c-programming/categories/:categoryId/questions", cProgrammingResController.CreateQuestion)
		admin.POST("/c-programming/resources/upload", cProgrammingResController.UploadResource)
		admin.GET("/c-programming/resources", cProgrammingResController.GetAdminResources)

		admin.GET("/resources/:id/content", cProgrammingResController.GetResourceCompleteContent)
		admin.POST("/resources/:id/videos", cProgrammingResController.AddVideoToResource)
		admin.POST("/resources/:id/articles", cProgrammingResController.AddArticleToResource)
		admin.POST("/resources/:id/exercise-categories", cProgrammingResController.CreateCategory)
		admin.POST("/exercise-categories/:categoryId/questions", cProgrammingResController.CreateQuestion)
		admin.GET("/c-programming/categories/:categoryId/questions/all", cProgrammingResController.AdminGetAllQuestionsByCategoryID)
		admin.PUT("/videos/:id", cProgrammingResController.UpdateVideo)
		admin.PUT("/articles/:id", cProgrammingResController.UpdateArticle)
		admin.PUT("/exercise-categories/:id", cProgrammingResController.UpdateExerciseCategory)
		admin.PUT("/questions/:id", cProgrammingResController.UpdateQuestion)
		admin.DELETE("/:itemType/:itemId", cProgrammingResController.DeleteContentItem)

		// 上传视频相关路由
		admin.POST("/upload/video", contentController.UploadVideo)
		admin.POST("/upload/video/chunk", contentController.UploadVideoChunk)
		admin.GET("/upload/video/progress/:uploadId", contentController.GetUploadProgress)
	}

	if cfg.Storage.Type == "local" {
		router.Static("/uploads", cfg.Storage.LocalPath)
		router.Static("/api/uploads", cfg.Storage.LocalPath)
	}

	return &App{
		Config: cfg,
		Router: router,
		DB:     db,
	}
}

func mustParseUint(s string) uint {
	id, _ := strconv.ParseUint(s, 10, 32)
	return uint(id)
}

func (a *App) Run() {
	log.Printf("Server running on port %s", a.Config.Server.Port)

	if err := a.Router.Run("127.0.0.1:" + a.Config.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
