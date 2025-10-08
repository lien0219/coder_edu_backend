package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"time"

	"gorm.io/gorm"
)

// LearningGoalService 处理学习目标的业务逻辑
type LearningGoalService struct {
	GoalRepo                    *repository.GoalRepository
	CProgrammingResourceRepo    *repository.CProgrammingResourceRepository
	CProgrammingResourceService *CProgrammingResourceService
	DB                          *gorm.DB
}

func NewLearningGoalService(
	goalRepo *repository.GoalRepository,
	cProgrammingResourceRepo *repository.CProgrammingResourceRepository,
	cProgrammingResourceService *CProgrammingResourceService,
	db *gorm.DB,
) *LearningGoalService {
	return &LearningGoalService{
		GoalRepo:                    goalRepo,
		CProgrammingResourceRepo:    cProgrammingResourceRepo,
		CProgrammingResourceService: cProgrammingResourceService,
		DB:                          db,
	}
}

// CreateGoalRequest 创建学习目标的请求结构
type CreateGoalRequest struct {
	Title            string    `json:"title" binding:"required,max=255"`
	Description      string    `json:"description" binding:"max=1000"`
	TargetDate       time.Time `json:"targetDate" binding:"required"`
	GoalType         string    `json:"goalType" binding:"required,oneof=short_term long_term"`
	ResourceModuleID uint      `json:"resourceModuleId" binding:"required"`
}

// UpdateGoalRequest 更新学习目标的请求结构
type UpdateGoalRequest struct {
	Title            string    `json:"title" binding:"max=255"`
	Description      string    `json:"description" binding:"max=1000"`
	TargetDate       time.Time `json:"targetDate"`
	GoalType         string    `json:"goalType" binding:"oneof=short_term long_term"`
	ResourceModuleID uint      `json:"resourceModuleId"`
}

// GetRecommendedResourceModules 获取推荐资源模块列表
func (s *LearningGoalService) GetRecommendedResourceModules() ([]model.CProgrammingResource, error) {
	// 获取所有启用的资源模块
	modules, _, err := s.CProgrammingResourceRepo.FindAll(1, 1000, "", &[]bool{true}[0], "order", "asc")
	if err != nil {
		return nil, err
	}
	return modules, nil
}

// CreateGoal 创建新的学习目标
func (s *LearningGoalService) CreateGoal(userID uint, req CreateGoalRequest) (*model.Goal, error) {
	// 验证资源模块是否存在
	resourceModule, err := s.CProgrammingResourceRepo.FindByID(req.ResourceModuleID)
	if err != nil {
		return nil, err
	}

	// 创建学习目标
	goal := &model.Goal{
		UserID:             userID,
		Title:              req.Title,
		Description:        req.Description,
		Status:             model.GoalPending,
		Current:            0,
		Target:             100,
		Progress:           0,
		TargetDate:         req.TargetDate,
		GoalType:           model.GoalType(req.GoalType),
		ResourceModuleID:   req.ResourceModuleID,
		ResourceModuleName: resourceModule.Name,
	}

	return goal, s.GoalRepo.Create(goal)
}

// GetUserGoals 获取用户的所有学习目标
func (s *LearningGoalService) GetUserGoals(userID uint) ([]model.Goal, error) {
	goals, err := s.GoalRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	// 更新每个目标的状态和进度
	for i := range goals {
		s.updateGoalStatusAndProgress(&goals[i], userID)
	}

	return goals, nil
}

// GetUserGoalsByType 获取用户特定类型的学习目标
func (s *LearningGoalService) GetUserGoalsByType(userID uint, goalType model.GoalType) ([]model.Goal, error) {
	goals, err := s.GoalRepo.FindByUserIDAndGoalType(userID, goalType)
	if err != nil {
		return nil, err
	}

	// 更新每个目标的状态和进度
	for i := range goals {
		s.updateGoalStatusAndProgress(&goals[i], userID)
	}

	return goals, nil
}

// GetGoalByID 获取特定ID的学习目标
func (s *LearningGoalService) GetGoalByID(userID, goalID uint) (*model.Goal, error) {
	goal, err := s.GoalRepo.FindByIDAndUserID(goalID, userID)
	if err != nil {
		return nil, err
	}

	// 更新目标的状态和进度
	s.updateGoalStatusAndProgress(goal, userID)

	return goal, nil
}

// UpdateGoal 更新学习目标
func (s *LearningGoalService) UpdateGoal(userID, goalID uint, req UpdateGoalRequest) (*model.Goal, error) {
	goal, err := s.GoalRepo.FindByIDAndUserID(goalID, userID)
	if err != nil {
		return nil, err
	}

	// 更新目标信息
	if req.Title != "" {
		goal.Title = req.Title
	}
	if req.Description != "" {
		goal.Description = req.Description
	}
	if !req.TargetDate.IsZero() {
		goal.TargetDate = req.TargetDate
	}
	if req.GoalType != "" {
		goal.GoalType = model.GoalType(req.GoalType)
	}
	if req.ResourceModuleID > 0 {
		// 验证资源模块是否存在
		resourceModule, err := s.CProgrammingResourceRepo.FindByID(req.ResourceModuleID)
		if err != nil {
			return nil, err
		}
		goal.ResourceModuleID = req.ResourceModuleID
		goal.ResourceModuleName = resourceModule.Name
	}

	// 更新目标的状态和进度
	s.updateGoalStatusAndProgress(goal, userID)

	return goal, s.GoalRepo.Update(goal)
}

// DeleteGoal 删除学习目标
func (s *LearningGoalService) DeleteGoal(userID, goalID uint) error {
	// 验证目标是否属于用户
	_, err := s.GoalRepo.FindByIDAndUserID(goalID, userID)
	if err != nil {
		return err
	}

	return s.GoalRepo.Delete(goalID)
}

// updateGoalStatusAndProgress 更新目标的状态和进度
func (s *LearningGoalService) updateGoalStatusAndProgress(goal *model.Goal, userID uint) {
	// 获取资源模块的进度
	resourceModuleProgress, err := s.CProgrammingResourceService.GetResourceModuleWithProgress(goal.ResourceModuleID, userID)
	if err != nil {
		return
	}

	// 更新目标进度
	goal.Progress = resourceModuleProgress.Progress
	goal.Current = int(resourceModuleProgress.Progress)

	// 检查是否已完成
	isCompleted := resourceModuleProgress.IsCompleted

	// 检查是否已过期
	today := time.Now()
	isExpired := !today.Before(goal.TargetDate)

	// 更新目标状态
	if isCompleted {
		if isExpired {
			goal.Status = model.GoalCompletedExpired
		} else {
			goal.Status = model.GoalCompleted
		}
	} else {
		if resourceModuleProgress.Progress > 0 {
			// 进行中
			if isExpired {
				goal.Status = model.GoalInProgressExpired
			} else {
				goal.Status = model.GoalInProgress
			}
		} else {
			// 未开始
			if isExpired {
				goal.Status = model.GoalPendingExpired
			} else {
				goal.Status = model.GoalPending
			}
		}
	}

	// 保存更新
	s.GoalRepo.Update(goal)
}
