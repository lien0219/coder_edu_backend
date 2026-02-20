package app

import (
	"coder_edu_backend/docs"
	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/middleware"
	"coder_edu_backend/internal/model"

	"coder_edu_backend/pkg/monitoring"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (a *App) registerRoutes(router *gin.Engine, c *controllers, repos *repositories, cfg *config.Config) {
	docs.SwaggerInfo.BasePath = "/api"
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")))

	router.GET("/metrics", monitoring.PrometheusHandler())

	// 1. 公共路由(无需登录)
	a.registerPublicRoutes(router, c)

	// 2. 社区模块
	a.registerCommunityRoutes(router, c, repos)

	// 3. 需要授权的路由
	authGroup := router.Group("/api")
	authGroup.Use(middleware.AuthMiddleware(cfg), middleware.ActivityMiddleware(repos.user))
	{
		// 学生/通用 授权接口
		a.registerStudentRoutes(authGroup, c)

		// 教师相关接口
		a.registerTeacherRoutes(authGroup, c)
	}

	// 4. 管理员相关接口
	a.registerAdminRoutes(router, c, repos, cfg)
}

func (a *App) registerCommunityRoutes(router *gin.Engine, c *controllers, repos *repositories) {
	community := router.Group("/api/community")
	community.Use(middleware.ActivityMiddleware(repos.user))
	{
		// 列表类：可选认证，允许游客访问，登录用户可看我的
		community.GET("/posts", middleware.TryAuthMiddleware(a.Config), c.community.GetPosts)
		community.GET("/posts/list", middleware.TryAuthMiddleware(a.Config), c.community.ListPosts)
		community.GET("/posts/:id", middleware.TryAuthMiddleware(a.Config), c.community.GetPostDetail)
		community.GET("/posts/:id/comments", middleware.TryAuthMiddleware(a.Config), c.community.GetPostComments)
		community.GET("/questions", middleware.TryAuthMiddleware(a.Config), c.community.GetQuestions)
		community.GET("/resources", middleware.TryAuthMiddleware(a.Config), c.community.GetResources)
		community.GET("/resources/:id", middleware.TryAuthMiddleware(a.Config), c.community.GetResourceDetail)

		// 交互类：强制认证
		authorized := community.Group("/")
		authorized.Use(middleware.AuthMiddleware(a.Config))
		{
			authorized.POST("/posts", c.community.CreatePost)
			authorized.PUT("/posts/:id", c.community.UpdatePost)
			authorized.DELETE("/posts/:id", c.community.DeletePost)
			authorized.POST("/posts/:id/comments", c.community.CreateComment)
			authorized.DELETE("/comments/:id", c.community.DeleteComment)
			authorized.POST("/questions", c.community.CreateQuestion)
			authorized.POST("/questions/:questionId/answers", c.community.AnswerQuestion)
			authorized.POST("/resources", c.community.CreateResource)
			authorized.POST("/resources/upload", c.community.UploadResourceFile)
			authorized.GET("/resources/:id/download", c.community.DownloadResource)
			authorized.DELETE("/resources/:id", c.community.DeleteResource)
			authorized.POST("/:type/:id/upvote", c.community.Upvote)
		}
	}
}

func (a *App) registerPublicRoutes(router *gin.Engine, c *controllers) {
	public := router.Group("/api")
	{
		public.GET("/health", c.health.HealthCheck)
		public.POST("/register", c.auth.Register)
		public.POST("/login", c.auth.Login)
		public.GET("/motivation", c.motivation.GetCurrentMotivation)

		// 验证码相关
		captcha := public.Group("/auth/captcha")
		{
			captcha.POST("/verify", c.auth.VerifyCaptcha)
			captcha.GET("/check-skip", c.auth.CheckCaptchaSkip)
		}
	}

	// 无需权限的答案提交接口
	publicAPI := router.Group("/api/public")
	{
		publicAPI.POST("/c-programming/questions/:questionId/submit", c.cProgramming.SubmitExerciseAnswerPublic)
	}
}

func (a *App) registerStudentRoutes(rg *gin.RouterGroup, c *controllers) {
	rg.GET("/profile", c.auth.GetProfile)
	rg.PUT("/user/profile", c.user.UpdateProfile)
	rg.POST("/user/avatar/upload", c.user.UploadAvatar)
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

	// 视频上传相关（通用）
	rg.POST("/upload/video", c.content.UploadVideo)
	rg.POST("/upload/video/chunk", c.content.UploadVideoChunk)
	rg.GET("/upload/video/progress/:uploadId", c.content.GetUploadProgress)

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

	// AI 问答
	rg.POST("/qa/ask", c.qa.Ask)
	rg.GET("/qa/history", c.qa.GetHistory)
	rg.GET("/qa/history/detail", c.qa.GetHistoryDetail)
	rg.DELETE("/qa/history/:sessionId", c.qa.DeleteSession) // 删除会话
	rg.GET("/qa/report/weekly", c.qa.GetWeeklyReport)       // 学习周报接口
	rg.POST("/qa/diagnose", c.qa.DiagnoseCode)              // 代码诊断接口

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

	// 课后测试
	rg.GET("/student/post-class-tests/published", c.postClassTest.GetPublishedTest)
	rg.GET("/student/post-class-tests/:id", c.postClassTest.GetStudentTestDetail)
	rg.POST("/student/post-class-tests/:id/start", c.postClassTest.StartTest)
	rg.POST("/student/post-class-tests/:id/learning-time", c.postClassTest.RecordLearningTime)
	rg.POST("/student/post-class-tests/:id/submit", c.postClassTest.SubmitTest)

	// 迁移任务
	rg.GET("/student/migration-tasks/published", c.migrationTask.GetPublishedTasks)
	rg.GET("/student/migration-tasks/:id", c.migrationTask.GetStudentTaskDetail)
	rg.POST("/student/migration-tasks/:id/start", c.migrationTask.StartTask)
	rg.POST("/student/migration-tasks/:id/submit", c.migrationTask.SubmitTask)
	rg.POST("/student/migration-tasks/:id/learning-time", c.migrationTask.RecordLearningTime)

	// 学前测试
	rg.GET("/assessments/questions", c.assessment.GetStudentQuestions)
	rg.POST("/assessments/submit", c.assessment.SubmitAssessment)
	rg.GET("/assessments/result", c.assessment.GetMyResult)

	// 学习路径
	rg.GET("/learning-path/student", c.learningPath.GetStudentPath)
	rg.GET("/learning-path/levels/:level/materials", c.learningPath.GetMaterialsByLevel)
	rg.POST("/learning-path/materials/:id/learning-time", c.learningPath.RecordLearningTime)
	rg.POST("/learning-path/materials/:id/complete", c.learningPath.CompleteMaterial)

	// 有效反思
	rg.GET("/reflections/my", c.reflection.GetMyReflection)
	rg.POST("/reflections/my", c.reflection.SaveMyReflection)

	// 协作中心 - 聊天室
	chat := rg.Group("/chat")
	{
		chat.GET("/overview", c.chat.GetOverview)
		chat.GET("/ws", c.chat.HandleWS)
		chat.GET("/conversations", c.chat.GetConversations)
		chat.POST("/groups", c.chat.CreateGroup)
		chat.POST("/privates", c.chat.CreatePrivateChat)
		chat.PUT("/conversations/:id", c.chat.UpdateGroupInfo)   // 修改群信息
		chat.DELETE("/conversations/:id", c.chat.DisbandGroup)   // 解散群聊
		chat.POST("/conversations/:id/leave", c.chat.LeaveGroup) // 退出群聊
		chat.GET("/conversations/:id/messages", c.chat.GetHistory)
		chat.GET("/messages/:id/context", c.chat.GetMessageContext) // 获取消息上下文
		chat.PUT("/messages/:id/revoke", c.chat.RevokeMessage)      // 撤回消息
		chat.GET("/conversations/:id/members", c.chat.GetMembers)
		chat.POST("/conversations/:id/members", c.chat.InviteMember)         // 邀请成员
		chat.DELETE("/conversations/:id/members/:userId", c.chat.KickMember) // 踢出成员
		chat.POST("/conversations/:id/transfer", c.chat.TransferAdmin)       // 转让群主
		chat.POST("/conversations/:id/messages", c.chat.SendMessage)
		chat.PUT("/conversations/:id/read", c.chat.MarkAsRead)
		chat.GET("/search", c.chat.GlobalSearch) // 全局搜索
		chat.POST("/upload", c.chat.UploadFile)

		chat.GET("/users/search", c.chat.SearchUser)
		chat.GET("/users/search-fuzzy", c.chat.SearchUsers)
		chat.GET("/friends", c.chat.GetFriends)
		chat.DELETE("/friends/:id", c.chat.DeleteFriend)
		chat.GET("/friend-requests", c.chat.GetFriendRequests)
		chat.POST("/friend-requests", c.chat.SendFriendRequest)
		chat.PUT("/friend-requests/:id", c.chat.HandleFriendRequest)
	}
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
		teacher.POST("/post-class-tests/submissions/reset", c.postClassTest.ResetStudentTests)

		// 迁移任务管理
		teacher.POST("/migration-tasks", c.migrationTask.CreateTask)
		teacher.GET("/migration-tasks", c.migrationTask.ListTasks)
		teacher.GET("/migration-tasks/:id", c.migrationTask.GetTask)
		teacher.PUT("/migration-tasks/:id", c.migrationTask.UpdateTask)
		teacher.DELETE("/migration-tasks/:id", c.migrationTask.DeleteTask)
		teacher.GET("/migration-tasks/:id/submissions", c.migrationTask.ListSubmissions)
		teacher.GET("/migration-tasks/submissions/:id", c.migrationTask.GetSubmissionDetail)

		// 有效反思管理
		teacher.GET("/reflections", middleware.RoleMiddleware(model.Teacher, model.Admin), c.reflection.ListAllReflections)
		teacher.PUT("/reflections/user/:userId", middleware.RoleMiddleware(model.Teacher, model.Admin), c.reflection.UpdateReflection)
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

func (a *App) registerAdminRoutes(router *gin.Engine, c *controllers, repos *repositories, cfg *config.Config) {
	admin := router.Group("/api/admin")
	admin.Use(middleware.AuthMiddleware(a.Config), middleware.ActivityMiddleware(repos.user))
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
		}
	}
}
