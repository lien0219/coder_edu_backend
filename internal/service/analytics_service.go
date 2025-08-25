package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"time"
)

type AnalyticsService struct {
	ProgressRepo       *repository.ProgressRepository
	SessionRepo        *repository.SessionRepository
	SkillRepo          *repository.SkillRepository
	LearningLogRepo    *repository.LearningLogRepository
	RecommendationRepo *repository.RecommendationRepository
}

func NewAnalyticsService(
	progressRepo *repository.ProgressRepository,
	sessionRepo *repository.SessionRepository,
	skillRepo *repository.SkillRepository,
	learningLogRepo *repository.LearningLogRepository,
	recommendationRepo *repository.RecommendationRepository,
) *AnalyticsService {
	return &AnalyticsService{
		ProgressRepo:       progressRepo,
		SessionRepo:        sessionRepo,
		SkillRepo:          skillRepo,
		LearningLogRepo:    learningLogRepo,
		RecommendationRepo: recommendationRepo,
	}
}

func (s *AnalyticsService) GetLearningOverview(userID uint) (*model.LearningOverview, error) {
	// 获取总体进度
	progress, err := s.ProgressRepo.GetOverallProgress(userID)
	if err != nil {
		return nil, err
	}

	// 获取月度数据
	monthlyData, err := s.ProgressRepo.GetMonthlyProgress(userID, 6) // 最近6个月
	if err != nil {
		return nil, err
	}

	// 获取模块完成情况
	moduleCompletion, err := s.ProgressRepo.GetModuleCompletion(userID)
	if err != nil {
		return nil, err
	}

	return &model.LearningOverview{
		TotalModules:     progress.TotalModules,
		CompletedModules: progress.CompletedModules,
		AverageScore:     progress.AverageScore,
		MonthlyProgress:  monthlyData,
		ModuleCompletion: moduleCompletion,
	}, nil
}

func (s *AnalyticsService) GetLearningProgress(userID uint, weeks int) (*model.LearningProgress, error) {
	weeklyData, err := s.ProgressRepo.GetWeeklyProgress(userID, weeks)
	if err != nil {
		return nil, err
	}

	// 计算趋势
	trend := "stable"
	if len(weeklyData) >= 2 {
		last := weeklyData[len(weeklyData)-1]
		secondLast := weeklyData[len(weeklyData)-2]

		if last.AverageScore > secondLast.AverageScore && last.ModulesCompleted >= secondLast.ModulesCompleted {
			trend = "improving"
		} else if last.AverageScore < secondLast.AverageScore && last.ModulesCompleted <= secondLast.ModulesCompleted {
			trend = "declining"
		}
	}

	return &model.LearningProgress{
		Weeks: weeklyData,
		Trend: trend,
	}, nil
}

func (s *AnalyticsService) GetSkillAssessments(userID uint) (*model.SkillRadar, error) {
	skills, err := s.SkillRepo.GetLatestAssessments(userID)
	if err != nil {
		return nil, err
	}

	skillNames := make([]string, len(skills))
	knowledgeCoverage := make([]int, len(skills))
	problemSolving := make([]int, len(skills))

	for i, skill := range skills {
		skillNames[i] = skill.Skill
		knowledgeCoverage[i] = skill.Score
		problemSolving[i] = skill.Score
	}

	return &model.SkillRadar{
		Skills:            skillNames,
		KnowledgeCoverage: knowledgeCoverage,
		ProblemSolving:    problemSolving,
	}, nil
}

func (s *AnalyticsService) GetPersonalizedRecommendations(userID uint) (*model.PersonalizedRecommendation, error) {
	// 基于用户学习数据生成个性化建议
	recommendations, err := s.RecommendationRepo.GenerateForUser(userID)
	if err != nil {
		return nil, err
	}

	return recommendations, nil
}

func (s *AnalyticsService) StartLearningSession(userID, moduleID uint) (uint, error) {
	session := &model.LearningSession{
		UserID:    userID,
		ModuleID:  moduleID,
		StartTime: time.Now(),
	}

	err := s.SessionRepo.Create(session)
	if err != nil {
		return 0, err
	}

	return session.ID, nil
}

func (s *AnalyticsService) EndLearningSession(userID, sessionID uint, activity string) error {
	session, err := s.SessionRepo.FindByIDAndUserID(sessionID, userID)
	if err != nil {
		return err
	}

	endTime := time.Now()
	duration := int(endTime.Sub(session.StartTime).Minutes())

	session.EndTime = &endTime
	session.Duration = duration
	session.Activity = activity

	return s.SessionRepo.Update(session)
}
