package app

import (
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
	"time"

	"coder_edu_backend/docs"

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
}

type services struct {
	auth                 *service.AuthService
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
	health         *controller.HealthController
}

func (a *App) RegisterConfigCallback(callback func(*config.Config)) {
	a.configCallbacks = append(a.configCallbacks, callback)
}

func (a *App) initRepositories(db *gorm.DB) *repositories {
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
	}
}

func (a *App) initServices(repos *repositories, cfg *config.Config, db *gorm.DB) *services {
	s := &services{}

	s.auth = service.NewAuthService(repos.user, cfg)
	s.content = service.NewContentService(repos.resource, cfg)
	s.motivation = service.NewMotivationService(repos.motivation)
	s.dashboard = service.NewDashboardService(repos.user, repos.task, repos.resource, repos.goal, s.motivation)
	s.learning = service.NewLearningService(repos.module, repos.task, repos.resource, repos.progress, repos.learningLog, repos.quiz, cfg, db)
	s.achievement = service.NewAchievementService(repos.achievement, repos.user, repos.goal)
	s.community = service.NewCommunityService(repos.post, nil, repos.question, repos.answer, repos.user)
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
	s.postClassTest = service.NewPostClassTestService(repos.postClassTest)

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
		user:           controller.NewUserController(s.user),
		cProgramming:   controller.NewCProgrammingResourceController(s.cProgrammingResource, s.content),
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

	router.Use(func(c *gin.Context) {
		c.Set("config", cfg)
		c.Next()
	})
}

func (a *App) registerRoutes(router *gin.Engine, c *controllers, repos *repositories, cfg *config.Config) {
	docs.SwaggerInfo.BasePath = "/api"
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")))

	router.GET("/metrics", monitoring.PrometheusHandler())

	// 1. 公共路由 (无需登录)
	a.registerPublicRoutes(router, c)

	// 2. 需要授权的路由
	authGroup := router.Group("/api")
	authGroup.Use(middleware.AuthMiddleware(), middleware.ActivityMiddleware(repos.user))
	{
		// 学生/通用 授权接口
		a.registerStudentRoutes(authGroup, c)

		// 教师相关接口
		a.registerTeacherRoutes(authGroup, c)
	}

	// 3. 管理员相关接口
	a.registerAdminRoutes(router, c)
}

func (a *App) registerPublicRoutes(router *gin.Engine, c *controllers) {
	public := router.Group("/api")
	{
		public.GET("/health", c.health.HealthCheck)
		public.POST("/register", c.auth.Register)
		public.POST("/login", c.auth.Login)
		public.GET("/motivation", c.motivation.GetCurrentMotivation)
	}

	// 无需权限的答案提交接口
	publicAPI := router.Group("/api/public")
	{
		publicAPI.POST("/c-programming/questions/:questionId/submit", c.cProgramming.SubmitExerciseAnswerPublic)
	}
}

func (a *App) registerStudentRoutes(rg *gin.RouterGroup, c *controllers) {
	rg.GET("/profile", c.auth.GetProfile)
	rg.GET("/resources", c.content.GetResources)
	rg.GET("/knowledge-tags", c.knowledgeTag.ListTags)
	rg.GET("/dashboard", c.dashboard.GetDashboard)
	rg.GET("/dashboard/today-tasks", c.dashboard.GetTodayTasks)
	rg.PATCH("/dashboard/tasks/:taskId", c.dashboard.UpdateTaskStatus)

	// 知识点相关
	rg.GET("/knowledge-points/student", c.knowledgePoint.ListForStudent)
	rg.GET("/knowledge-points/ranking", c.knowledgePoint.GetRanking)
	rg.GET("/knowledge-points/student/:id", c.knowledgePoint.GetDetailForStudent)
	rg.POST("/knowledge-points/student/:id/start", c.knowledgePoint.StartExercises)
	rg.POST("/knowledge-points/student/submit", c.knowledgePoint.SubmitExercises)
	rg.POST("/knowledge-points/student/:id/learning-time", c.knowledgePoint.RecordLearningTime)

	// 学习相关
	rg.GET("/learning/pre-class", c.learning.GetPreClass)
	rg.GET("/learning/in-class", c.learning.GetInClass)
	rg.GET("/learning/post-class", c.learning.GetPostClass)
	rg.POST("/learning/learning-log", c.learning.SubmitLearningLog)
	rg.POST("/learning/quiz/:quizId", c.learning.SubmitQuiz)
	rg.POST("/learning/run-code", c.learning.ExecuteCode)

	// 成就/目标
	rg.GET("/achievements", c.achievement.GetUserAchievements)
	rg.GET("/achievements/leaderboard", c.achievement.GetLeaderboard)
	rg.GET("/achievements/goals", c.achievement.GetUserGoals)
	rg.POST("/achievements/goals", c.achievement.CreateGoal)
	rg.PATCH("/achievements/goals/:goalId", c.achievement.UpdateGoalProgress)

	// 社区
	rg.GET("/community/posts", c.community.GetPosts)
	rg.POST("/community/posts", c.community.CreatePost)
	rg.GET("/community/questions", c.community.GetQuestions)
	rg.POST("/community/questions", c.community.CreateQuestion)
	rg.POST("/community/questions/:questionId/answers", c.community.AnswerQuestion)
	rg.POST("/community/:type/:id/upvote", c.community.Upvote)

	// 分析
	rg.GET("/analytics/overview", c.analytics.GetOverview)
	rg.GET("/analytics/progress", c.analytics.GetProgress)
	rg.GET("/analytics/challenges/weekly", c.analytics.GetWeeklyChallengeStats)
	rg.GET("/analytics/skills", c.analytics.GetSkills)
	rg.GET("/analytics/abilities", c.analytics.GetAbilities)
	rg.GET("/analytics/levels/:levelId/curve", c.analytics.GetLevelCurve)
	rg.GET("/analytics/recommendations", c.analytics.GetRecommendations)
	rg.POST("/analytics/session/start", c.analytics.StartSession)
	rg.POST("/analytics/session/:sessionId/end", c.analytics.EndSession)

	// 关卡挑战
	rg.GET("/levels/student", c.level.GetStudentLevels)
	rg.GET("/levels/student/:id", c.level.GetStudentLevelDetail)
	rg.GET("/levels/student/:id/questions", c.level.GetStudentLevelQuestions)
	rg.GET("/levels/basic-info", c.level.GetAllLevelsBasicInfo)
	rg.POST("/levels/:id/attempts/start", c.level.StartAttempt)
	rg.POST("/levels/:id/attempts/:attemptId/submit", c.level.BatchSubmitAnswers)
	rg.POST("/attempts/:id/submit", c.level.SubmitAttempt)
	rg.GET("/levels/ranking", c.level.GetLevelRanking)
	rg.GET("/users/:userId/level-total-score", c.level.GetUserLevelTotalScore)
	rg.GET("/users/:userId/level-stats", c.level.GetUserLevelStats)

	// C语言资源
	rg.GET("/c-programming/resources", c.cProgramming.GetResources)
	rg.GET("/c-programming/resources/full", c.cProgramming.GetResourcesWithAllContent)
	rg.GET("/c-programming/resources/:id", c.cProgramming.GetResourceByID)
	rg.GET("/c-programming/resources/:id/categories", c.cProgramming.GetCategoriesByResourceID)
	rg.GET("/c-programming/categories/:categoryId/questions", c.cProgramming.GetQuestionsByCategoryID)
	rg.GET("/c-programming/categories/:categoryId/questions-with-status", c.cProgramming.GetQuestionsByCategoryIDWithUserStatus)
	rg.GET("/c-programming/resources/:id/videos", c.cProgramming.GetVideosByResourceID)
	rg.GET("/c-programming/resources/:id/articles", c.cProgramming.GetArticlesByResourceID)
	rg.GET("/c-programming/exercises/users/:userID/questions/:questionID/submission", c.cProgramming.CheckUserSubmittedQuestion)

	// 用户相关
	rg.POST("/users/checkin", c.user.Checkin)
	rg.GET("/users/checkin/stats", c.user.GetCheckinStats)
	rg.GET("/users/stats", c.user.GetUserStats)
	rg.GET("/users/level-status", c.user.GetLevelStatus)
	rg.POST("/users/:id/points", middleware.RoleMiddleware(model.Student, model.Teacher, model.Admin), c.user.UpdateUserPoints)

	// 资源进度
	rg.GET("/c-programming/resource-progress/:resourceId", c.cProgramming.GetResourceModuleWithProgress)
	rg.POST("/c-programming/resource-progress/:resourceId/completion", c.cProgramming.UpdateResourceCompletionStatus)
	rg.GET("/c-programming/resource-progress/unfinished", c.cProgramming.GetUnfinishedResourceModules)
	rg.GET("/c-programming/resource-progress/all", c.cProgramming.GetAllResourceModulesWithProgress)

	// 学习目标
	rg.GET("/learning-goals/resources", c.learningGoal.GetRecommendedResourceModules)
	rg.GET("/learning-goals", c.learningGoal.GetUserGoals)
	rg.GET("/learning-goals/type", c.learningGoal.GetUserGoalsByType)
	rg.POST("/learning-goals", c.learningGoal.CreateGoal)
	rg.GET("/learning-goals/:id", c.learningGoal.GetGoalByID)
	rg.PUT("/learning-goals/:id", c.learningGoal.UpdateGoal)
	rg.DELETE("/learning-goals/:id", c.learningGoal.DeleteGoal)
	rg.GET("/learning-goals/:id/details", c.learningGoal.GetGoalDetails)

	// 任务相关
	rg.GET("/tasks/today", c.task.GetTodayTasks)
	rg.POST("/tasks/:taskItemId/completion", c.task.UpdateTaskCompletion)

	// 教师建议
	rg.GET("/suggestions", c.suggestion.ListStudentSuggestions)
	rg.POST("/suggestions/:id/complete", c.suggestion.CompleteSuggestion)

	// 学前测试
	rg.GET("/assessments/questions", c.assessment.GetStudentQuestions)
	rg.POST("/assessments/submit", c.assessment.SubmitAssessment)
	rg.GET("/assessments/result", c.assessment.GetMyResult)

	// 学习路径
	rg.GET("/learning-path/student", c.learningPath.GetStudentPath)
	rg.GET("/learning-path/levels/:level/materials", c.learningPath.GetMaterialsByLevel)
	rg.POST("/learning-path/materials/:id/learning-time", c.learningPath.RecordLearningTime)
	rg.POST("/learning-path/materials/:id/complete", c.learningPath.CompleteMaterial)
}

func (a *App) registerTeacherRoutes(rg *gin.RouterGroup, c *controllers) {
	teacher := rg.Group("/teacher")
	teacher.Use(middleware.RoleMiddleware(model.Teacher, model.Admin, model.Student))
	{
		// 周任务
		teacher.POST("/tasks/weekly", c.task.SetWeeklyTask)
		teacher.GET("/tasks/weekly", c.task.GetWeeklyTasks)
		teacher.GET("/tasks/weekly/current", c.task.GetCurrentWeekTask)
		teacher.DELETE("/tasks/weekly/:taskId", c.task.DeleteWeeklyTask)

		// 关卡管理
		teacher.POST("/levels", c.level.CreateLevel)
		teacher.GET("/levels", c.level.ListLevels)
		teacher.GET("/levels/:id", c.level.GetLevel)
		teacher.PUT("/levels/:id", c.level.UpdateLevel)
		teacher.DELETE("/levels/:id", c.level.DeleteLevel)
		teacher.POST("/levels/:id/publish", c.level.PublishLevel)
		teacher.POST("/levels/bulk/publish", c.level.BulkPublish)
		teacher.POST("/levels/bulk", c.level.BulkUpdate)
		teacher.GET("/levels/:id/versions", c.level.GetVersions)
		teacher.POST("/levels/:id/versions/:versionId/rollback", c.level.RollbackVersion)

		// 题目管理
		teacher.POST("/levels/:id/questions", c.level.CreateQuestion)
		teacher.PUT("/levels/:id/questions/:qid", c.level.UpdateQuestion)
		teacher.DELETE("/levels/:id/questions/:qid", c.level.DeleteQuestion)

		// 评分相关
		teacher.GET("/levels/:id/attempts/pending-grading", c.grade.ListPendingGrading)
		teacher.POST("/levels/:id/attempts/:attemptId/grade", c.grade.GradeAttempt)

		// 学生进度
		teacher.GET("/students/progress", c.suggestion.ListStudentsProgress)
		teacher.GET("/students/:id/progress", c.suggestion.GetStudentProgress)

		// 尝试统计
		teacher.GET("/levels/:id/attempts/stats", c.level.GetAttemptStats)
		teacher.POST("/levels/:id/attempts/start", c.level.StartAttempt)
		teacher.POST("/levels/:id/attempts/:attemptId/submit", c.level.SubmitAttempt)

		// 可见性与排期
		teacher.PUT("/levels/:id/visibility", c.level.UpdateVisibility)
		teacher.POST("/levels/:id/schedule_publish", c.level.SchedulePublish)

		// 建议管理
		teacher.POST("/suggestions", c.suggestion.CreateSuggestion)
		teacher.PUT("/suggestions/:id", c.suggestion.UpdateSuggestion)
		teacher.GET("/suggestions", c.suggestion.ListTeacherSuggestions)
		teacher.DELETE("/suggestions/:id", c.suggestion.DeleteSuggestion)

		// 学前测试管理
		teacher.POST("/assessments", c.assessment.CreateAssessment)
		teacher.GET("/assessments", c.assessment.ListAssessments)
		teacher.GET("/assessments/:id", c.assessment.GetAssessment)
		teacher.POST("/assessments/questions", c.assessment.CreateQuestion)
		teacher.GET("/assessments/questions", c.assessment.ListQuestions)
		teacher.GET("/assessments/questions/:id", c.assessment.GetQuestion)
		teacher.PUT("/assessments/questions/:id", c.assessment.UpdateQuestion)
		teacher.DELETE("/assessments/questions/:id", c.assessment.DeleteQuestion)

		// 提交管理
		teacher.GET("/assessments/submissions", c.assessment.ListSubmissions)
		teacher.GET("/assessments/submissions/:id", c.assessment.GetSubmissionDetail)
		teacher.POST("/assessments/submissions/:id/grade", c.assessment.GradeSubmission)
		teacher.DELETE("/assessments/submissions/:id", c.assessment.DeleteSubmission)
		teacher.POST("/assessments/retest", c.assessment.SetUserRetest)

		// 知识点管理
		teacher.POST("/knowledge-points", c.knowledgePoint.Create)
		teacher.GET("/knowledge-points", c.knowledgePoint.List)
		teacher.PUT("/knowledge-points/:id", c.knowledgePoint.Update)
		teacher.DELETE("/knowledge-points/:id", c.knowledgePoint.Delete)
		teacher.GET("/knowledge-points/points-list", c.knowledgePoint.GetStudentsPointsList)
		teacher.POST("/knowledge-points/reward", c.knowledgePoint.RewardStudents)

		// 知识点审核
		teacher.GET("/knowledge-points/submissions", c.knowledgePoint.ListSubmissions)
		teacher.GET("/knowledge-points/submissions/:id", c.knowledgePoint.GetSubmissionDetail)
		teacher.POST("/knowledge-points/submissions/:id/audit", c.knowledgePoint.AuditSubmission)

		// 课后测试试卷管理
		teacher.POST("/post-class-tests", c.postClassTest.CreateTest)
		teacher.GET("/post-class-tests", c.postClassTest.ListTests)
		teacher.GET("/post-class-tests/:id", c.postClassTest.GetTest)
		teacher.PUT("/post-class-tests/:id", c.postClassTest.UpdateTest)
		teacher.DELETE("/post-class-tests/:id", c.postClassTest.DeleteTest)

		// 课后测试答题管理
		teacher.GET("/post-class-tests/:id/submissions", c.postClassTest.ListSubmissions)
		teacher.GET("/post-class-tests/submissions/:id", c.postClassTest.GetSubmissionDetail)
		teacher.POST("/post-class-tests/submissions/:id/reset", c.postClassTest.ResetStudentTest)
		teacher.POST("/post-class-tests/submissions/batch-reset", c.postClassTest.BatchResetStudentTests)
	}

	// 学习路径管理
	learningPath := rg.Group("/teacher/learning-path")
	learningPath.Use(middleware.RoleMiddleware(model.Teacher, model.Admin))
	{
		learningPath.POST("/materials", c.learningPath.CreateMaterial)
		learningPath.GET("/materials", c.learningPath.ListMaterials)
		learningPath.GET("/materials/:id", c.learningPath.GetMaterial)
		learningPath.PUT("/materials/:id", c.learningPath.UpdateMaterial)
		learningPath.DELETE("/materials/:id", c.learningPath.DeleteMaterial)
	}
}

func (a *App) registerAdminRoutes(router *gin.Engine, c *controllers) {
	admin := router.Group("/api/admin")
	admin.Use(middleware.AuthMiddleware())
	{
		// 1. 用户列表和详情：允许管理员和老师访问
		admin.GET("/users", middleware.RoleMiddleware(model.Admin, model.Teacher), c.user.GetUsers)
		admin.GET("/users/:id", middleware.RoleMiddleware(model.Admin, model.Teacher), c.user.GetUser)

		// 2. 其他所有接口：仅限管理员访问
		adminOnly := admin.Group("/")
		adminOnly.Use(middleware.RoleMiddleware(model.Admin))
		{
			adminOnly.POST("/upload/icon", c.content.UploadIcon)
			adminOnly.POST("/resources", c.content.UploadResource)
			adminOnly.PUT("/users/:id", c.user.UpdateUser)
			adminOnly.DELETE("/users/:id", c.user.DeleteUser)
			adminOnly.POST("/users/:id/reset-password", c.user.ResetPassword)
			adminOnly.POST("/users/:id/disable", c.user.DisableUser)

			adminOnly.GET("/motivations", c.motivation.GetAllMotivations)
			adminOnly.POST("/motivations", c.motivation.CreateMotivation)
			adminOnly.PUT("/motivations/:id", c.motivation.UpdateMotivation)
			adminOnly.DELETE("/motivations/:id", c.motivation.DeleteMotivation)
			adminOnly.POST("/motivations/:id/switch", c.motivation.SwitchMotivation)

			adminOnly.POST("/c-programming/resources", c.cProgramming.CreateResource)
			adminOnly.PUT("/c-programming/resources/:id", c.cProgramming.UpdateResource)
			adminOnly.DELETE("/c-programming/resources/:id", c.cProgramming.DeleteResource)
			adminOnly.POST("/c-programming/resources/:id/categories", c.cProgramming.CreateCategory)
			adminOnly.POST("/c-programming/categories/:categoryId/questions", c.cProgramming.CreateQuestion)
			adminOnly.POST("/c-programming/resources/upload", c.cProgramming.UploadResource)
			adminOnly.GET("/c-programming/resources", c.cProgramming.GetAdminResources)

			adminOnly.GET("/resources/:id/content", c.cProgramming.GetResourceCompleteContent)
			adminOnly.POST("/resources/:id/videos", c.cProgramming.AddVideoToResource)
			adminOnly.POST("/resources/:id/articles", c.cProgramming.AddArticleToResource)
			adminOnly.POST("/resources/:id/exercise-categories", c.cProgramming.CreateCategory)
			adminOnly.POST("/exercise-categories/:categoryId/questions", c.cProgramming.CreateQuestion)
			adminOnly.GET("/c-programming/categories/:categoryId/questions/all", c.cProgramming.AdminGetAllQuestionsByCategoryID)
			adminOnly.PUT("/videos/:id", c.cProgramming.UpdateVideo)
			adminOnly.PUT("/articles/:id", c.cProgramming.UpdateArticle)
			adminOnly.PUT("/exercise-categories/:id", c.cProgramming.UpdateExerciseCategory)
			adminOnly.PUT("/questions/:id", c.cProgramming.UpdateQuestion)
			adminOnly.DELETE("/:itemType/:itemId", c.cProgramming.DeleteContentItem)

			// 上传视频相关路由
			adminOnly.POST("/upload/video", c.content.UploadVideo)
			adminOnly.POST("/upload/video/chunk", c.content.UploadVideoChunk)
			adminOnly.GET("/upload/video/progress/:uploadId", c.content.GetUploadProgress)
		}
	}
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

	app := &App{
		Config: cfg,
		DB:     db,
	}

	repos := app.initRepositories(db)
	services := app.initServices(repos, cfg, db)
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

	app.startBackgroundTasks(services)

	return app
}

func (a *App) Run() {
	log.Printf("Server running on port %s", a.Config.Server.Port)

	if err := a.Router.Run("127.0.0.1:" + a.Config.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
