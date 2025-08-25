package service

import (
	"coder_edu_backend/internal/model"

	"coder_edu_backend/internal/repository"
)

type AchievementService struct {
	AchievementRepo *repository.AchievementRepository
	UserRepo        *repository.UserRepository
	GoalRepo        *repository.GoalRepository
}

func NewAchievementService(
	achievementRepo *repository.AchievementRepository,
	userRepo *repository.UserRepository,
	goalRepo *repository.GoalRepository,
) *AchievementService {
	return &AchievementService{
		AchievementRepo: achievementRepo,
		UserRepo:        userRepo,
		GoalRepo:        goalRepo,
	}
}

type UserAchievements struct {
	TotalXP      int                 `json:"totalXp"`
	CurrentLevel int                 `json:"currentLevel"`
	NextLevelXP  int                 `json:"nextLevelXp"`
	Badges       []model.Achievement `json:"badges"`
	Leaderboard  []LeaderboardEntry  `json:"leaderboard"`
	Goals        []model.Goal        `json:"goals"`
}

type LeaderboardEntry struct {
	Rank   int    `json:"rank"`
	User   string `json:"user"`
	XP     int    `json:"xp"`
	Avatar string `json:"avatar,omitempty"`
}

type GoalRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	TargetValue int    `json:"targetValue" binding:"required"`
}

func (s *AchievementService) GetUserAchievements(userID uint) (*UserAchievements, error) {
	// 获取用户信息
	user, err := s.UserRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	// 获取用户成就
	achievements, err := s.AchievementRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	// 获取排行榜
	leaderboard, err := s.GetLeaderboard(10)
	if err != nil {
		return nil, err
	}

	// 获取用户目标
	goals, err := s.GoalRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	// 计算等级
	level, nextLevelXP := calculateLevel(user.XP)

	return &UserAchievements{
		TotalXP:      user.XP,
		CurrentLevel: level,
		NextLevelXP:  nextLevelXP,
		Badges:       achievements,
		Leaderboard:  leaderboard,
		Goals:        goals,
	}, nil
}

func (s *AchievementService) GetLeaderboard(limit int) ([]LeaderboardEntry, error) {
	users, err := s.UserRepo.FindTopByXP(limit)
	if err != nil {
		return nil, err
	}

	leaderboard := make([]LeaderboardEntry, len(users))
	for i, user := range users {
		leaderboard[i] = LeaderboardEntry{
			Rank:   i + 1,
			User:   user.Name,
			XP:     user.XP,
			Avatar: "", // 可以添加头像URL
		}
	}

	return leaderboard, nil
}

func (s *AchievementService) GetUserGoals(userID uint) ([]model.Goal, error) {
	return s.GoalRepo.FindByUserID(userID)
}

func (s *AchievementService) CreateGoal(userID uint, req GoalRequest) (*model.Goal, error) {
	goal := &model.Goal{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Target:      req.TargetValue,
		Current:     0,
	}

	err := s.GoalRepo.Create(goal)
	if err != nil {
		return nil, err
	}

	return goal, nil
}

func (s *AchievementService) UpdateGoalProgress(userID uint, goalID uint, progress int) error {
	goal, err := s.GoalRepo.FindByIDAndUserID(goalID, userID)
	if err != nil {
		return err
	}

	goal.Current = progress
	if progress >= 100 {
		goal.Status = model.GoalCompleted

		// 奖励XP
		xpReward := 50 // 根据目标难度可以调整
		err = s.UserRepo.UpdateXP(userID, xpReward)
		if err != nil {
			return err
		}
	}

	return s.GoalRepo.Update(goal)
}

func calculateLevel(xp int) (int, int) {
	// 简单等级计算：每200XP升一级
	level := xp / 200
	nextLevelXP := (level + 1) * 200
	return level, nextLevelXP
}
