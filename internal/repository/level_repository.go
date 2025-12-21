package repository

import (
	"coder_edu_backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type LevelRepository struct {
	DB *gorm.DB
}

func NewLevelRepository(db *gorm.DB) *LevelRepository {
	return &LevelRepository{DB: db}
}

func (r *LevelRepository) Create(level *model.Level) error {
	return r.DB.Create(level).Error
}

func (r *LevelRepository) Update(level *model.Level) error {
	return r.DB.Save(level).Error
}

func (r *LevelRepository) FindByID(id uint) (*model.Level, error) {
	var level model.Level
	err := r.DB.First(&level, id).Error
	return &level, err
}

func (r *LevelRepository) ListByCreator(creatorID uint, page, limit int) ([]model.Level, int, error) {
	var levels []model.Level
	var total int64
	query := r.DB.Model(&model.Level{})
	if creatorID > 0 {
		query = query.Where("creator_id = ?", creatorID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * limit
	err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&levels).Error
	return levels, int(total), err
}

func (r *LevelRepository) BulkUpdate(ids []uint, updates map[string]interface{}) error {
	if len(ids) == 0 {
		return nil
	}
	return r.DB.Model(&model.Level{}).Where("id IN ?", ids).Updates(updates).Error
}

func (r *LevelRepository) CreateVersion(version *model.LevelVersion) error {
	return r.DB.Create(version).Error
}

func (r *LevelRepository) GetVersions(levelID uint) ([]model.LevelVersion, error) {
	var versions []model.LevelVersion
	err := r.DB.Where("level_id = ?", levelID).Order("version_number desc").Find(&versions).Error
	return versions, err
}

func (r *LevelRepository) GetVersionByID(id uint) (*model.LevelVersion, error) {
	var v model.LevelVersion
	err := r.DB.First(&v, id).Error
	return &v, err
}

func (r *LevelRepository) DeleteQuestionsByLevel(levelID uint) error {
	return r.DB.Where("level_id = ?", levelID).Delete(&model.LevelQuestion{}).Error
}

func (r *LevelRepository) CreateQuestions(questions []model.LevelQuestion) error {
	if len(questions) == 0 {
		return nil
	}
	return r.DB.Create(&questions).Error
}

func (r *LevelRepository) CreateQuestion(question *model.LevelQuestion) error {
	return r.DB.Create(question).Error
}

func (r *LevelRepository) UpdateQuestion(question *model.LevelQuestion) error {
	return r.DB.Save(question).Error
}

func (r *LevelRepository) FindQuestionByID(id uint) (*model.LevelQuestion, error) {
	var q model.LevelQuestion
	if err := r.DB.First(&q, id).Error; err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *LevelRepository) DeleteQuestionByID(id uint) error {
	return r.DB.Delete(&model.LevelQuestion{}, id).Error
}

func (r *LevelRepository) CountAttemptsByUserLevel(userID, levelID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&model.LevelAttempt{}).Where("user_id = ? AND level_id = ?", userID, levelID).Count(&count).Error
	return count, err
}

func (r *LevelRepository) GetQuestionsByLevel(levelID uint) ([]model.LevelQuestion, error) {
	var qs []model.LevelQuestion
	err := r.DB.Where("level_id = ?", levelID).Order("`order` asc").Find(&qs).Error
	return qs, err
}

func (r *LevelRepository) DeleteQuestionsByLevelID(levelID uint) error {
	return r.DB.Where("level_id = ?", levelID).Delete(&model.LevelQuestion{}).Error
}

func (r *LevelRepository) DeleteLevelAbilitiesByLevelID(levelID uint) error {
	return r.DB.Where("level_id = ?", levelID).Delete(&model.LevelAbility{}).Error
}

func (r *LevelRepository) DeleteLevelKnowledgeByLevelID(levelID uint) error {
	return r.DB.Where("level_id = ?", levelID).Delete(&model.LevelKnowledge{}).Error
}

func (r *LevelRepository) UpdateLevel(level *model.Level) error {
	return r.DB.Save(level).Error
}

// BulkSetPublished 批量设置发布状态与发布时间
func (r *LevelRepository) BulkSetPublished(ids []uint, publish bool, publishAt *time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	updates := map[string]interface{}{"is_published": publish}
	if publish {
		if publishAt != nil {
			updates["published_at"] = *publishAt
		} else {
			updates["published_at"] = time.Now()
		}
	} else {
		updates["published_at"] = nil
	}
	return r.DB.Model(&model.Level{}).Where("id IN ?", ids).Updates(updates).Error
}
