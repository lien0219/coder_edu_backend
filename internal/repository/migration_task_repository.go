package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type MigrationTaskRepository struct {
	DB *gorm.DB
}

func NewMigrationTaskRepository(db *gorm.DB) *MigrationTaskRepository {
	return &MigrationTaskRepository{DB: db}
}

func (r *MigrationTaskRepository) CreateTask(task *model.MigrationTask) error {
	return r.DB.Create(task).Error
}

func (r *MigrationTaskRepository) FindTaskByID(id string) (*model.MigrationTask, error) {
	var task model.MigrationTask
	err := r.DB.First(&task, "id = ?", id).Error
	return &task, err
}

func (r *MigrationTaskRepository) UpdateTask(task *model.MigrationTask) error {
	return r.DB.Save(task).Error
}

func (r *MigrationTaskRepository) DeleteTask(id string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ?", id).Delete(&model.MigrationQuestion{}).Error; err != nil {
			return err
		}
		var submissionIDs []string
		if err := tx.Model(&model.MigrationSubmission{}).Where("task_id = ?", id).Pluck("id", &submissionIDs).Error; err == nil && len(submissionIDs) > 0 {
			if err := tx.Where("submission_id IN ?", submissionIDs).Delete(&model.MigrationAnswer{}).Error; err != nil {
				return err
			}
			if err := tx.Where("task_id = ?", id).Delete(&model.MigrationSubmission{}).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&model.MigrationTask{}, "id = ?", id).Error
	})
}

type MigrationTaskListRow struct {
	model.MigrationTask
	QuestionCount  int `json:"questionCount"`
	CompletedCount int `json:"completedCount"`
}

func (r *MigrationTaskRepository) ListTasks(page, limit int) ([]MigrationTaskListRow, int64, error) {
	var total int64
	query := r.DB.Model(&model.MigrationTask{}).Where("deleted_at IS NULL")
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tasks []MigrationTaskListRow
	dbQuery := r.DB.Table("migration_tasks t").
		Select("t.*, " +
			"(SELECT COUNT(*) FROM migration_questions q WHERE q.task_id = t.id AND q.deleted_at IS NULL) as question_count, " +
			"(SELECT COUNT(*) FROM migration_submissions s JOIN users u ON s.user_id = u.id WHERE s.task_id = t.id AND s.deleted_at IS NULL AND s.status = 'completed' AND u.deleted_at IS NULL AND u.disabled = 0) as completed_count").
		Where("t.deleted_at IS NULL")

	if limit > 0 {
		offset := (page - 1) * limit
		dbQuery = dbQuery.Offset(offset).Limit(limit)
	}

	err := dbQuery.Order("t.created_at desc").Scan(&tasks).Error
	return tasks, total, err
}

func (r *MigrationTaskRepository) CreateQuestion(question *model.MigrationQuestion) error {
	return r.DB.Create(question).Error
}

func (r *MigrationTaskRepository) FindQuestionByID(id string) (*model.MigrationQuestion, error) {
	var q model.MigrationQuestion
	err := r.DB.First(&q, "id = ?", id).Error
	return &q, err
}

func (r *MigrationTaskRepository) UpdateQuestion(question *model.MigrationQuestion) error {
	return r.DB.Save(question).Error
}

func (r *MigrationTaskRepository) DeleteQuestion(id string) error {
	return r.DB.Delete(&model.MigrationQuestion{}, "id = ?", id).Error
}

func (r *MigrationTaskRepository) ListQuestions(taskID string) ([]model.MigrationQuestion, error) {
	var qs []model.MigrationQuestion
	err := r.DB.Where("task_id = ?", taskID).Order("`order` asc, created_at desc").Find(&qs).Error
	return qs, err
}

func (r *MigrationTaskRepository) UpdateSubmissionWithAnswers(submission *model.MigrationSubmission, answers []model.MigrationAnswer) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(submission).Error; err != nil {
			return err
		}
		for i := range answers {
			answers[i].SubmissionID = submission.ID
		}
		if len(answers) > 0 {
			if err := tx.Create(&answers).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *MigrationTaskRepository) FindSubmissionByUserAndTask(userID uint, taskID string) (*model.MigrationSubmission, error) {
	var s model.MigrationSubmission
	err := r.DB.Where("user_id = ? AND task_id = ?", userID, taskID).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *MigrationTaskRepository) ListSubmissions(taskID string, page, limit int, studentName string, status string) ([]map[string]interface{}, int64, error) {
	var total int64
	query := r.DB.Table("migration_submissions s").
		Select("s.*, u.name as user_name, u.email as user_email, t.title as task_title").
		Joins("JOIN users u ON s.user_id = u.id").
		Joins("JOIN migration_tasks t ON s.task_id = t.id").
		Where("s.deleted_at IS NULL AND u.deleted_at IS NULL AND u.disabled = ? AND t.deleted_at IS NULL", false)

	if taskID != "" && taskID != "all" {
		query = query.Where("s.task_id = ?", taskID)
	}

	if studentName != "" {
		query = query.Where("u.name LIKE ?", "%"+studentName+"%")
	}

	if status != "" && status != "all" {
		query = query.Where("s.status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var results []map[string]interface{}
	offset := (page - 1) * limit
	err := query.Order("s.created_at desc").Offset(offset).Limit(limit).Scan(&results).Error
	return results, total, err
}

func (r *MigrationTaskRepository) GetSubmissionDetail(submissionID string) (*model.MigrationSubmission, []model.MigrationAnswer, error) {
	var submission model.MigrationSubmission
	if err := r.DB.First(&submission, "id = ?", submissionID).Error; err != nil {
		return nil, nil, err
	}

	var answers []model.MigrationAnswer
	if err := r.DB.Where("submission_id = ?", submissionID).Find(&answers).Error; err != nil {
		return nil, nil, err
	}

	return &submission, answers, nil
}

func (r *MigrationTaskRepository) ListPublishedTasksForStudent(userID uint) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.DB.Table("migration_tasks t").
		Select("t.*, "+
			"(SELECT COUNT(*) FROM migration_questions q WHERE q.task_id = t.id AND q.deleted_at IS NULL) as question_count, "+
			"COALESCE(s.status, 'pending') as status, s.score, s.completed_at").
		Joins("LEFT JOIN migration_submissions s ON s.task_id = t.id AND s.user_id = ? AND s.deleted_at IS NULL", userID).
		Where("t.is_published = ? AND t.deleted_at IS NULL", true).
		Order("t.published_at desc, t.created_at desc").
		Scan(&results).Error
	return results, err
}
