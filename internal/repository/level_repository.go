package repository

import (
	"coder_edu_backend/internal/model"
	"fmt"
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

func (r *LevelRepository) GetAllLevelsBasicInfo() ([]model.Level, error) {
	var levels []model.Level
	err := r.DB.Model(&model.Level{}).Select("id, title").Order("id asc").Find(&levels).Error
	return levels, err
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

func (r *LevelRepository) DeleteLevel(id uint) error {
	tx := r.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 删除尝试答案记录
	if err := tx.Where("attempt_id IN (SELECT id FROM level_attempts WHERE level_id = ?)", id).Delete(&model.LevelAttemptAnswer{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 2. 删除尝试题目分数记录
	if err := tx.Where("attempt_id IN (SELECT id FROM level_attempts WHERE level_id = ?)", id).Delete(&model.LevelAttemptQuestionScore{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 3. 删除尝试题目时间记录
	if err := tx.Where("attempt_id IN (SELECT id FROM level_attempts WHERE level_id = ?)", id).Delete(&model.LevelAttemptQuestionTime{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 4. 删除尝试记录
	if err := tx.Where("level_id = ?", id).Delete(&model.LevelAttempt{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 5. 删除关卡题目
	if err := tx.Where("level_id = ?", id).Delete(&model.LevelQuestion{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 6. 删除关卡能力要求
	if err := tx.Where("level_id = ?", id).Delete(&model.LevelAbility{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 7. 删除关卡知识点
	if err := tx.Where("level_id = ?", id).Delete(&model.LevelKnowledge{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 8. 删除关卡版本记录
	if err := tx.Where("level_id = ?", id).Delete(&model.LevelVersion{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 9. 最后删除关卡本身
	if err := tx.Delete(&model.Level{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *LevelRepository) ListLevelsForStudent(userID uint, search string, difficulty string, page, limit int) ([]model.Level, int, error) {
	var levels []model.Level
	var total int64

	query := r.DB.Model(&model.Level{}).Where("is_published = ?", true)

	// 可见性筛选
	query = query.Where("visible_scope = ? OR (visible_scope = ? AND JSON_CONTAINS(visible_to, CAST(? AS CHAR)))",
		"all", "specific", userID)

	// 时间范围筛选
	now := time.Now()
	query = query.Where("visible_scope = ? OR ((available_from IS NULL OR available_from <= ?) AND (available_to IS NULL OR available_to >= ?))",
		"all", now, now)

	// 搜索条件
	if search != "" {
		query = query.Where("title LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 难度筛选
	if difficulty != "" && difficulty != "all" {
		query = query.Where("difficulty = ?", difficulty)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询并预加载题目
	offset := (page - 1) * limit
	err := query.Preload("Questions").Order("created_at desc").Offset(offset).Limit(limit).Find(&levels).Error
	return levels, int(total), err
}

func (r *LevelRepository) GetAttemptsByUserAndLevels(userID uint, levelIDs []uint) ([]model.LevelAttempt, error) {
	var attempts []model.LevelAttempt
	err := r.DB.Where("user_id = ? AND level_id IN ?", userID, levelIDs).Find(&attempts).Error
	return attempts, err
}

func (r *LevelRepository) GetLevelAbilities(levelID uint) ([]model.LevelAbility, error) {
	var abilities []model.LevelAbility
	err := r.DB.Where("level_id = ?", levelID).Find(&abilities).Error
	return abilities, err
}

func (r *LevelRepository) GetLevelKnowledge(levelID uint) ([]model.LevelKnowledge, error) {
	var knowledge []model.LevelKnowledge
	err := r.DB.Where("level_id = ?", levelID).Find(&knowledge).Error
	return knowledge, err
}

func (r *LevelRepository) CreateAttempt(attempt *model.LevelAttempt) error {
	return r.DB.Create(attempt).Error
}

func (r *LevelRepository) FindAttemptByID(id uint) (*model.LevelAttempt, error) {
	var attempt model.LevelAttempt
	err := r.DB.First(&attempt, id).Error
	return &attempt, err
}

func (r *LevelRepository) UpdateAttempt(attempt *model.LevelAttempt) error {
	return r.DB.Save(attempt).Error
}

func (r *LevelRepository) CreateAttemptAnswers(answers []model.LevelAttemptAnswer) error {
	if len(answers) == 0 {
		return nil
	}
	return r.DB.Create(&answers).Error
}

func (r *LevelRepository) CreateAttemptQuestionTimes(times []model.LevelAttemptQuestionTime) error {
	if len(times) == 0 {
		return nil
	}
	return r.DB.Create(&times).Error
}

func (r *LevelRepository) GetAttemptStats(levelID uint, start *time.Time, end *time.Time, studentID uint) (int64, float64, float64, int64, error) {
	query := r.DB.Model(&model.LevelAttempt{}).
		Joins("JOIN users ON users.id = level_attempts.user_id").
		Where("level_attempts.level_id = ? AND level_attempts.deleted_at IS NULL", levelID)

	if studentID == 0 {
		query = query.Where("users.disabled = ?", false)
	} else {
		query = query.Where("level_attempts.user_id = ?", studentID)
	}

	if start != nil {
		query = query.Where("level_attempts.started_at >= ?", *start)
	}
	if end != nil {
		query = query.Where("level_attempts.started_at <= ?", *end)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, 0, 0, 0, err
	}

	var avgScore float64
	var avgTime float64
	var successCount int64
	if total > 0 {
		if err := query.Select("AVG(score)").Scan(&avgScore).Error; err != nil {
			return 0, 0, 0, 0, err
		}
		if err := query.Select("AVG(total_time_seconds)").Scan(&avgTime).Error; err != nil {
			return 0, 0, 0, 0, err
		}
		if err := query.Select("SUM(success)").Scan(&successCount).Error; err != nil {
			return 0, 0, 0, 0, err
		}
	}
	return total, avgScore, avgTime, successCount, nil
}

func (r *LevelRepository) GetLevelRanking(limit int) ([]model.LevelRankingEntry, error) {
	query := `
		WITH user_level_best_scores AS (
			SELECT
				la.user_id,
				la.level_id,
				MAX(la.score) as best_score
			FROM level_attempts la
			WHERE la.success = true AND la.deleted_at IS NULL
			GROUP BY la.user_id, la.level_id
		),
		user_stats AS (
			SELECT
				u.id as user_id,
				u.name as username,
				SUM(ulbs.best_score) as total_score,
				MAX(ulbs.best_score) as max_score
			FROM users u
			INNER JOIN user_level_best_scores ulbs ON u.id = ulbs.user_id
			WHERE u.role = 'student' AND u.deleted_at IS NULL AND u.disabled = false
			GROUP BY u.id, u.name
			HAVING SUM(ulbs.best_score) > 0
		),
		user_best_levels AS (
			SELECT
				us.user_id,
				us.username,
				us.total_score,
				l.title as best_level_title,
				ROW_NUMBER() OVER (PARTITION BY us.user_id ORDER BY ulbs.best_score DESC) as rn
			FROM user_stats us
			INNER JOIN user_level_best_scores ulbs ON us.user_id = ulbs.user_id AND ulbs.best_score = us.max_score
			INNER JOIN levels l ON ulbs.level_id = l.id
		)
		SELECT
			ROW_NUMBER() OVER (ORDER BY total_score DESC, user_id ASC) as ranking,
			username,
			best_level_title,
			total_score
		FROM user_best_levels
		WHERE rn = 1
		ORDER BY total_score DESC, user_id ASC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	var rankings []model.LevelRankingEntry
	err := r.DB.Raw(query).Scan(&rankings).Error
	return rankings, err
}

func (r *LevelRepository) GetUserLevelTotalScore(userID uint) (int, error) {
	query := `
		WITH user_level_best_scores AS (
			SELECT
				la.user_id,
				la.level_id,
				MAX(la.score) as best_score
			FROM level_attempts la
			WHERE la.success = true AND la.user_id = ? AND la.deleted_at IS NULL
			GROUP BY la.user_id, la.level_id
		)
		SELECT COALESCE(SUM(best_score), 0) as total_score
		FROM user_level_best_scores
	`

	var totalScore int
	err := r.DB.Raw(query, userID).Scan(&totalScore).Error
	return totalScore, err
}

func (r *LevelRepository) GetUserWeeklyTimeHours(userID uint) (float64, error) {
	query := `
		SELECT COALESCE(SUM(total_time_seconds) / 3600.0, 0) as weekly_time_hours
		FROM level_attempts
		WHERE user_id = ?
			AND YEARWEEK(started_at, 1) = YEARWEEK(NOW(), 1)
			AND ended_at IS NOT NULL
			AND deleted_at IS NULL
	`
	var hours float64
	err := r.DB.Raw(query, userID).Scan(&hours).Error
	return hours, err
}

func (r *LevelRepository) GetUserAverageSuccessRate(userID uint) (float64, error) {
	query := `
		SELECT
			CASE
				WHEN COUNT(*) = 0 THEN 0
				ELSE ROUND((SUM(CASE WHEN success = true THEN 1 ELSE 0 END) * 100.0) / COUNT(*), 2)
			END as success_rate
		FROM level_attempts
		WHERE user_id = ? AND ended_at IS NOT NULL AND deleted_at IS NULL
	`
	var rate float64
	err := r.DB.Raw(query, userID).Scan(&rate).Error
	return rate, err
}

func (r *LevelRepository) GetUserSolvedChallengesCount(userID uint) (int, error) {
	query := `
		SELECT COUNT(DISTINCT level_id) as solved_count
		FROM level_attempts
		WHERE user_id = ? AND success = true AND deleted_at IS NULL
	`
	var count int
	err := r.DB.Raw(query, userID).Scan(&count).Error
	return count, err
}

func (r *LevelRepository) GetLevelAbilitiesWithDetails(levelID uint) ([]model.Ability, error) {
	var abilities []model.Ability
	err := r.DB.Table("abilities").
		Joins("JOIN level_abilities ON level_abilities.ability_id = abilities.id").
		Where("level_abilities.level_id = ?", levelID).
		Find(&abilities).Error
	return abilities, err
}

func (r *LevelRepository) GetLevelKnowledgeWithDetails(levelID uint) ([]model.KnowledgeTag, error) {
	var tags []model.KnowledgeTag
	err := r.DB.Table("knowledge_tags").
		Joins("JOIN level_knowledge ON level_knowledge.knowledge_tag_id = knowledge_tags.id").
		Where("level_knowledge.level_id = ?", levelID).
		Find(&tags).Error
	return tags, err
}

func (r *LevelRepository) GetCreator(creatorID uint) (*model.User, error) {
	var user model.User
	err := r.DB.First(&user, creatorID).Error
	return &user, err
}
