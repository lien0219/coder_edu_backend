package app

import (
	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/controller"
	"coder_edu_backend/internal/repository"
	"coder_edu_backend/internal/service"
	"coder_edu_backend/pkg/database"
	"coder_edu_backend/pkg/logger"
	"coder_edu_backend/pkg/monitoring"
	"coder_edu_backend/pkg/security"
	"coder_edu_backend/pkg/tracing"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	Config          *config.Config
	Router          *gin.Engine
	DB              *gorm.DB
	Redis           *redis.Client
	services        *services
	configCallbacks []func(*config.Config)
}

type repositories struct {
	user               *repository.UserRepository
	resource           *repository.ResourceRepository
	task               *repository.TaskRepository
	goal               *repository.GoalRepository
	module             *repository.ModuleRepository
	progress           *repository.ProgressRepository
	learningLog        *repository.LearningLogRepository
	quiz               *repository.QuizRepository
	achievement        *repository.AchievementRepository
	post               *repository.PostRepository
	comment            *repository.CommentRepository
	question           *repository.QuestionRepository
	answer             *repository.AnswerRepository
	session            *repository.SessionRepository
	skill              *repository.SkillRepository
	recommendation     *repository.RecommendationRepository
	motivation         *repository.MotivationRepository
	cProgrammingRes    *repository.CProgrammingResourceRepository
	exerciseCategory   *repository.ExerciseCategoryRepository
	exerciseQuestion   *repository.ExerciseQuestionRepository
	exerciseSubmission *repository.ExerciseSubmissionRepository
	checkin            *repository.CheckinRepository
	resourceCompletion *repository.ResourceCompletionRepository
	level              *repository.LevelRepository
	levelAttempt       *repository.LevelAttemptRepository
	knowledgeTag       *repository.KnowledgeTagRepository
	suggestion         *repository.SuggestionRepository
	assessment         *repository.AssessmentRepository
	learningPath       *repository.LearningPathRepository
	postClassTest      *repository.PostClassTestRepository
	migrationTask      *repository.MigrationTaskRepository
	reflection         *repository.ReflectionRepository
	chat               *repository.ChatRepository
	friendship         *repository.FriendshipRepository
	communityResource  *repository.CommunityResourceRepository
}

type services struct {
	auth                 *service.AuthService
	storage              *service.StorageService
	content              *service.ContentService
	motivation           *service.MotivationService
	dashboard            *service.DashboardService
	learning             *service.LearningService
	achievement          *service.AchievementService
	community            *service.CommunityService
	analytics            *service.AnalyticsService
	user                 *service.UserService
	task                 *service.TaskService
	cProgrammingResource *service.CProgrammingResourceService
	level                *service.LevelService
	knowledgeTag         *service.KnowledgeTagService
	suggestion           *service.SuggestionService
	assessment           *service.AssessmentService
	learningPath         *service.LearningPathService
	knowledgePoint       *service.KnowledgePointService
	learningGoal         *service.LearningGoalService
	postClassTest        *service.PostClassTestService
	migrationTask        *service.MigrationTaskService
	reflection           *service.ReflectionService
	chat                 *service.ChatService
	friendship           *service.FriendshipService
	chatHub              *service.ChatHub
}

type controllers struct {
	auth           *controller.AuthController
	content        *controller.ContentController
	motivation     *controller.MotivationController
	dashboard      *controller.DashboardController
	learning       *controller.LearningController
	achievement    *controller.AchievementController
	community      *controller.CommunityController
	analytics      *controller.AnalyticsController
	user           *controller.UserController
	cProgramming   *controller.CProgrammingResourceController
	learningGoal   *controller.LearningGoalController
	task           *controller.TaskController
	level          *controller.LevelController
	grade          *controller.GradeController
	suggestion     *controller.SuggestionController
	assessment     *controller.AssessmentController
	learningPath   *controller.LearningPathController
	knowledgePoint *controller.KnowledgePointController
	knowledgeTag   *controller.KnowledgeTagController
	postClassTest  *controller.PostClassTestController
	migrationTask  *controller.MigrationTaskController
	reflection     *controller.ReflectionController
	chat           *controller.ChatController
	health         *controller.HealthController
}

func (a *App) RegisterConfigCallback(callback func(*config.Config)) {
	a.configCallbacks = append(a.configCallbacks, callback)
}

func (a *App) initRepositories(db *gorm.DB, rdb *redis.Client) *repositories {
	return &repositories{
		user:               repository.NewUserRepository(db),
		resource:           repository.NewResourceRepository(db),
		task:               repository.NewTaskRepository(db),
		goal:               repository.NewGoalRepository(db),
		module:             repository.NewModuleRepository(db),
		progress:           repository.NewProgressRepository(db),
		learningLog:        repository.NewLearningLogRepository(db),
		quiz:               repository.NewQuizRepository(db),
		achievement:        repository.NewAchievementRepository(db),
		post:               repository.NewPostRepository(db),
		comment:            repository.NewCommentRepository(db),
		question:           repository.NewQuestionRepository(db),
		answer:             repository.NewAnswerRepository(db),
		session:            repository.NewSessionRepository(db),
		skill:              repository.NewSkillRepository(db),
		recommendation:     repository.NewRecommendationRepository(db),
		motivation:         repository.NewMotivationRepository(db),
		cProgrammingRes:    repository.NewCProgrammingResourceRepository(db),
		exerciseCategory:   repository.NewExerciseCategoryRepository(db),
		exerciseQuestion:   repository.NewExerciseQuestionRepository(db),
		exerciseSubmission: repository.NewExerciseSubmissionRepository(db),
		checkin:            repository.NewCheckinRepository(db),
		resourceCompletion: repository.NewResourceCompletionRepository(db),
		level:              repository.NewLevelRepository(db),
		levelAttempt:       repository.NewLevelAttemptRepository(db),
		knowledgeTag:       repository.NewKnowledgeTagRepository(db),
		suggestion:         repository.NewSuggestionRepository(db),
		assessment:         repository.NewAssessmentRepository(db),
		learningPath:       repository.NewLearningPathRepository(db),
		postClassTest:      repository.NewPostClassTestRepository(db),
		migrationTask:      repository.NewMigrationTaskRepository(db),
		reflection:         repository.NewReflectionRepository(db),
		chat:               repository.NewChatRepository(db, rdb),
		friendship:         repository.NewFriendshipRepository(db, rdb),
		communityResource:  repository.NewCommunityResourceRepository(db),
	}
}

func (a *App) initServices(repos *repositories, cfg *config.Config, db *gorm.DB, rdb *redis.Client) *services {
	s := &services{}

	s.storage = service.NewStorageService(cfg)
	s.auth = service.NewAuthService(repos.user, cfg)
	s.content = service.NewContentService(repos.resource, s.storage, cfg, rdb)
	s.motivation = service.NewMotivationService(repos.motivation)
	s.dashboard = service.NewDashboardService(repos.user, repos.task, repos.resource, repos.goal, s.motivation)
	s.learning = service.NewLearningService(repos.module, repos.task, repos.resource, repos.progress, repos.learningLog, repos.quiz, cfg, db)
	s.achievement = service.NewAchievementService(repos.achievement, repos.user, repos.goal)
	s.community = service.NewCommunityService(repos.post, repos.comment, repos.question, repos.answer, repos.user, repos.communityResource, rdb, cfg, s.storage)
	s.analytics = service.NewAnalyticsService(repos.progress, repos.session, repos.skill, repos.learningLog, repos.recommendation, repos.levelAttempt, db)
	s.user = service.NewUserServiceWithDB(repos.user, repos.checkin, db)

	s.task = service.NewTaskService(
		repos.task,
		repos.resource,
		repos.exerciseQuestion,
		repos.cProgrammingRes,
		repos.goal,
	)

	s.cProgrammingResource = service.NewCProgrammingResourceService(
		repos.cProgrammingRes,
		repos.exerciseCategory,
		repos.exerciseQuestion,
		repos.exerciseSubmission,
		repos.resource,
		repos.resourceCompletion,
		repos.goal,
		repos.task,
		s.task,
		db,
	)

	s.level = service.NewLevelService(repos.level, db)
	s.knowledgeTag = service.NewKnowledgeTagService(repos.knowledgeTag)
	s.suggestion = service.NewSuggestionService(repos.suggestion, repos.level, repos.levelAttempt)
	s.assessment = service.NewAssessmentService(repos.assessment)
	s.learningPath = service.NewLearningPathService(repos.learningPath, repos.assessment, repos.learningLog, repos.user)
	s.knowledgePoint = service.NewKnowledgePointService(db)
	s.learningGoal = service.NewLearningGoalService(
		repos.goal,
		repos.cProgrammingRes,
		s.cProgrammingResource,
		db,
	)
	s.postClassTest = service.NewPostClassTestService(repos.postClassTest, s.user)
	s.migrationTask = service.NewMigrationTaskService(repos.migrationTask, s.user)
	s.reflection = service.NewReflectionService(repos.reflection)

	s.chatHub = service.NewChatHub(rdb, repos.chat, repos.friendship)
	go s.chatHub.Run()

	s.chat = service.NewChatService(repos.chat)
	s.friendship = service.NewFriendshipService(repos.friendship, repos.user)

	return s
}

func (a *App) initControllers(s *services, db *gorm.DB) *controllers {
	return &controllers{
		auth:           controller.NewAuthController(s.auth, s.user),
		content:        controller.NewContentController(s.content),
		motivation:     controller.NewMotivationController(s.motivation),
		dashboard:      controller.NewDashboardController(s.dashboard),
		learning:       controller.NewLearningController(s.learning),
		achievement:    controller.NewAchievementController(s.achievement),
		community:      controller.NewCommunityController(s.community),
		analytics:      controller.NewAnalyticsController(s.analytics),
		user:           controller.NewUserController(s.user, s.storage, a.Config),
		cProgramming:   controller.NewCProgrammingResourceController(s.cProgrammingResource, s.content, a.Config),
		learningGoal:   controller.NewLearningGoalController(s.learningGoal),
		task:           controller.NewTaskController(s.task),
		level:          controller.NewLevelController(s.level, s.content),
		grade:          controller.NewGradeController(s.level),
		suggestion:     controller.NewSuggestionController(s.suggestion),
		assessment:     controller.NewAssessmentController(s.assessment),
		learningPath:   controller.NewLearningPathController(s.learningPath),
		knowledgePoint: controller.NewKnowledgePointController(s.knowledgePoint),
		knowledgeTag:   controller.NewKnowledgeTagController(s.knowledgeTag),
		postClassTest:  controller.NewPostClassTestController(s.postClassTest),
		migrationTask:  controller.NewMigrationTaskController(s.migrationTask),
		reflection:     controller.NewReflectionController(s.reflection),
		chat:           controller.NewChatController(s.chat, s.friendship, s.chatHub, s.storage, a.Config),
		health:         controller.NewHealthController(db),
	}
}

func (a *App) setupMiddlewares(router *gin.Engine, cfg *config.Config) {
	router.Use(security.CORS())
	router.Use(security.Secure())
	router.Use(security.RateLimiter(100000, time.Minute)) // 每分钟100000次请求

	// 分布式追踪中间件
	if cfg.Tracing.Enabled {
		router.Use(tracing.GinMiddleware())
	}

	router.Use(monitoring.MetricsMiddleware())
}

func (a *App) startBackgroundTasks(s *services) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		for range ticker.C {
			if err := s.level.ProcessScheduledPublishes(); err != nil {
				logger.Log.Error("scheduled publish error", zap.Error(err))
			}
		}
	}()
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

	rdb, err := database.InitRedis(&cfg.Redis)
	if err != nil {
		logger.Log.Fatal("Failed to initialize redis", zap.Error(err))
		log.Fatalf("Failed to initialize redis: %v", err)
	}

	app := &App{
		Config: cfg,
		DB:     db,
		Redis:  rdb,
	}

	repos := app.initRepositories(db, rdb)
	services := app.initServices(repos, cfg, db, rdb)
	app.services = services
	controllers := app.initControllers(services, db)

	// 监控初始化
	monitoring.Init()

	router := gin.Default()
	app.Router = router

	app.setupMiddlewares(router, cfg)

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
	}

	app.registerRoutes(router, controllers, repos, cfg)

	if cfg.Storage.Type == "local" {
		router.Static("/uploads", cfg.Storage.LocalPath)
		router.Static("/api/uploads", cfg.Storage.LocalPath)
	}

	// 社区资源文件存放路径
	if _, err := os.Stat("resource_file"); os.IsNotExist(err) {
		os.MkdirAll("resource_file", os.ModePerm)
	}
	router.Static("/api/community/resources/files", "resource_file")

	app.startBackgroundTasks(services)

	return app
}

func (a *App) Run() {
	srv := &http.Server{
		Addr:    ":" + a.Config.Server.Port,
		Handler: a.Router,
	}

	// 启动服务器
	go func() {
		log.Printf("Server running on port %s", a.Config.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 等待中断信号优雅地关闭服务器（设置5秒的超时时间）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 清理 WebSocket连接和Redis在线状态
	if a.services != nil && a.services.chatHub != nil {
		a.services.chatHub.Stop()
	}

	// 关闭服务
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
