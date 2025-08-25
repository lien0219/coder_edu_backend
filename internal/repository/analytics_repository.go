package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type SessionRepository struct {
	DB *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{DB: db}
}

func (r *SessionRepository) Create(session *model.LearningSession) error {
	return r.DB.Create(session).Error
}

func (r *SessionRepository) FindByIDAndUserID(sessionID, userID uint) (*model.LearningSession, error) {
	var session model.LearningSession
	err := r.DB.Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) Update(session *model.LearningSession) error {
	return r.DB.Save(session).Error
}

type SkillRepository struct {
	DB *gorm.DB
}

func NewSkillRepository(db *gorm.DB) *SkillRepository {
	return &SkillRepository{DB: db}
}

func (r *SkillRepository) GetLatestAssessments(userID uint) ([]model.SkillAssessment, error) {
	var skills []model.SkillAssessment

	// 获取每个技能的最新评估
	subquery := r.DB.Model(&model.SkillAssessment{}).
		Select("skill, MAX(assessed_at) as max_date").
		Where("user_id = ?", userID).
		Group("skill")

	err := r.DB.Joins("INNER JOIN (?) AS latest ON skill_assessments.skill = latest.skill AND skill_assessments.assessed_at = latest.max_date", subquery).
		Where("skill_assessments.user_id = ?", userID).
		Find(&skills).Error

	if err != nil {
		return nil, err
	}

	return skills, nil
}

type RecommendationRepository struct {
	DB *gorm.DB
}

func NewRecommendationRepository(db *gorm.DB) *RecommendationRepository {
	return &RecommendationRepository{DB: db}
}

func (r *RecommendationRepository) GenerateForUser(userID uint) (*model.PersonalizedRecommendation, error) {
	// 基于用户数据生成个性化建议
	// 模拟数据
	return &model.PersonalizedRecommendation{
		TimeManagement: "发现您的学习效率在午后有所下降，尝试将更多的任务安排在精力充沛的时刻。",
		FocusAreas: []string{
			"深入理解指针：根据分析显示在指针概念上仍存在困难，建议进行相关练习。",
			"复习数据结构：在链表的掌握上有所欠缺，建议加强练习。",
		},
		CommunitySuggestions: []string{
			"活跃的社区互动能有效提升学习效率，尝试在论坛中提出问题或帮助他人解决问题。",
		},
		ReviewTopics: []string{
			"指针概念",
			"链表实现",
			"内存管理",
		},
		ChallengeTasks: []string{
			"参与算法挑战",
			"完成一个完整的C语言项目",
		},
	}, nil
}
