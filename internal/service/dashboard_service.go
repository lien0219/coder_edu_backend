package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"time"
)

type DashboardService struct {
	UserRepo          *repository.UserRepository
	TaskRepo          *repository.TaskRepository
	ResourceRepo      *repository.ResourceRepository
	GoalRepo          *repository.GoalRepository
	MotivationService *MotivationService
}

func NewDashboardService(
	userRepo *repository.UserRepository,
	taskRepo *repository.TaskRepository,
	resourceRepo *repository.ResourceRepository,
	goalRepo *repository.GoalRepository,
	motivationService *MotivationService,
) *DashboardService {
	return &DashboardService{
		UserRepo:          userRepo,
		TaskRepo:          taskRepo,
		ResourceRepo:      resourceRepo,
		GoalRepo:          goalRepo,
		MotivationService: motivationService,
	}
}

type Dashboard struct {
	TodayTasks      []*model.Task       `json:"todayTasks"`
	GoalProgress    []GoalProgress      `json:"goalProgress"`
	Achievements    []model.Achievement `json:"achievements"`
	Recommended     []model.Resource    `json:"recommendedResources"`
	LearningStats   LearningStats       `json:"learningStats"`
	DailyMotivation string              `json:"dailyMotivation"`
}

type GoalProgress struct {
	Title    string  `json:"title"`
	Progress float64 `json:"progress"`
	Target   string  `json:"targetDate"`
}

type LearningStats struct {
	StudyTime        int     `json:"studyTime"`
	Accuracy         float64 `json:"accuracy"`
	Reflections      int     `json:"reflections"`
	ConceptsMastered int     `json:"conceptsMastered"`
}

func (s *DashboardService) GetUserDashboard(userID uint) (*Dashboard, error) {
	// 获取今日任务
	tasks, err := s.GetTodayTasks(userID)
	if err != nil {
		return nil, err
	}

	// 获取目标进度
	goals, err := s.GoalRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	goalProgress := make([]GoalProgress, len(goals))
	for i, goal := range goals {
		goalProgress[i] = GoalProgress{
			Title:    goal.Title,
			Progress: float64(goal.Progress),
			Target:   goal.TargetDate.Format("2006-01-02"),
		}
	}

	// 获取用户成就
	achievements, err := s.UserRepo.GetAchievements(userID)
	if err != nil {
		return nil, err
	}

	// 获取推荐资源
	resources, err := s.ResourceRepo.FindRecommended(userID, 10)
	if err != nil {
		return nil, err
	}

	// 获取学习统计
	stats, err := s.getLearningStats(userID)
	if err != nil {
		return nil, err
	}

	// 每日激励语
	dailyMotivation, err := s.MotivationService.GetCurrentMotivation()
	if err != nil || dailyMotivation == "" {
		dailyMotivation = "Every line of code you write is a step closer to mastery. Keep coding!"
	}

	return &Dashboard{
		TodayTasks:      tasks,
		GoalProgress:    goalProgress,
		Achievements:    achievements,
		Recommended:     resources,
		LearningStats:   stats,
		DailyMotivation: dailyMotivation,
	}, nil
}

func (s *DashboardService) GetTodayTasks(userID uint) ([]*model.Task, error) {
	today := time.Now()
	return s.TaskRepo.FindByUserAndDate(userID, today)
}

func (s *DashboardService) UpdateTaskStatus(taskID uint, status model.TaskStatus) error {
	return s.TaskRepo.UpdateStatus(taskID, status)
}

func (s *DashboardService) getLearningStats(userID uint) (LearningStats, error) {
	// 模拟数据
	return LearningStats{
		StudyTime:        15,
		Accuracy:         88.5,
		Reflections:      35,
		ConceptsMastered: 24,
	}, nil
}
