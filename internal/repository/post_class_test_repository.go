package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type PostClassTestRepository struct {
	DB *gorm.DB
}

func NewPostClassTestRepository(db *gorm.DB) *PostClassTestRepository {
	return &PostClassTestRepository{DB: db}
}

func (r *PostClassTestRepository) CreateTest(test *model.PostClassTest) error {
	return r.DB.Create(test).Error
}

func (r *PostClassTestRepository) FindTestByID(id string) (*model.PostClassTest, error) {
	var test model.PostClassTest
	err := r.DB.First(&test, "id = ?", id).Error
	return &test, err
}

func (r *PostClassTestRepository) UpdateTest(test *model.PostClassTest) error {
	return r.DB.Save(test).Error
}

func (r *PostClassTestRepository) DeleteTest(id string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("test_id = ?", id).Delete(&model.PostClassTestQuestion{}).Error; err != nil {
			return err
		}
		var submissionIDs []string
		if err := tx.Model(&model.PostClassTestSubmission{}).Where("test_id = ?", id).Pluck("id", &submissionIDs).Error; err == nil && len(submissionIDs) > 0 {
			if err := tx.Where("submission_id IN ?", submissionIDs).Delete(&model.PostClassTestAnswer{}).Error; err != nil {
				return err
			}
			if err := tx.Where("test_id = ?", id).Delete(&model.PostClassTestSubmission{}).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&model.PostClassTest{}, "id = ?", id).Error
	})
}

type PostClassTestListRow struct {
	model.PostClassTest
	QuestionCount  int `json:"questionCount"`
	CompletedCount int `json:"completedCount"`
}

func (r *PostClassTestRepository) ListTests(page, limit int) ([]PostClassTestListRow, int64, error) {
	var total int64
	query := r.DB.Model(&model.PostClassTest{}).Where("deleted_at IS NULL")
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tests []PostClassTestListRow
	dbQuery := r.DB.Table("post_class_tests t").
		Select("t.*, " +
			"(SELECT COUNT(*) FROM post_class_test_questions q WHERE q.test_id = t.id AND q.deleted_at IS NULL) as question_count, " +
			"(SELECT COUNT(*) FROM post_class_test_submissions s WHERE s.test_id = t.id AND s.deleted_at IS NULL AND s.status = 'completed') as completed_count").
		Where("t.deleted_at IS NULL")

	if limit > 0 && limit != 100 {
		offset := (page - 1) * limit
		dbQuery = dbQuery.Offset(offset).Limit(limit)
	}

	err := dbQuery.Order("t.created_at desc").Scan(&tests).Error
	return tests, total, err
}

func (r *PostClassTestRepository) CreateQuestion(question *model.PostClassTestQuestion) error {
	return r.DB.Create(question).Error
}

func (r *PostClassTestRepository) FindQuestionByID(id string) (*model.PostClassTestQuestion, error) {
	var q model.PostClassTestQuestion
	err := r.DB.First(&q, "id = ?", id).Error
	return &q, err
}

func (r *PostClassTestRepository) UpdateQuestion(question *model.PostClassTestQuestion) error {
	return r.DB.Save(question).Error
}

func (r *PostClassTestRepository) DeleteQuestion(id string) error {
	return r.DB.Delete(&model.PostClassTestQuestion{}, "id = ?", id).Error
}

func (r *PostClassTestRepository) ListQuestions(testID string) ([]model.PostClassTestQuestion, error) {
	var qs []model.PostClassTestQuestion
	err := r.DB.Where("test_id = ?", testID).Order("`order` asc, created_at desc").Find(&qs).Error
	return qs, err
}

func (r *PostClassTestRepository) ListSubmissions(testID string, page, limit int, studentName string) ([]map[string]interface{}, int64, error) {
	var total int64
	query := r.DB.Table("post_class_test_submissions s").
		Select("s.*, u.name as user_name, u.email as user_email").
		Joins("JOIN users u ON s.user_id = u.id").
		Where("s.test_id = ? AND s.deleted_at IS NULL", testID)

	if studentName != "" {
		query = query.Where("u.name LIKE ?", "%"+studentName+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var results []map[string]interface{}
	offset := (page - 1) * limit
	err := query.Order("s.created_at desc").Offset(offset).Limit(limit).Scan(&results).Error
	return results, total, err
}

func (r *PostClassTestRepository) GetSubmissionDetail(submissionID string) (*model.PostClassTestSubmission, []model.PostClassTestAnswer, error) {
	var submission model.PostClassTestSubmission
	if err := r.DB.First(&submission, "id = ?", submissionID).Error; err != nil {
		return nil, nil, err
	}

	var answers []model.PostClassTestAnswer
	if err := r.DB.Where("submission_id = ?", submissionID).Find(&answers).Error; err != nil {
		return nil, nil, err
	}

	return &submission, answers, nil
}

func (r *PostClassTestRepository) DeleteSubmission(submissionID string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("submission_id = ?", submissionID).Delete(&model.PostClassTestAnswer{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.PostClassTestSubmission{}, "id = ?", submissionID).Error
	})
}

func (r *PostClassTestRepository) BatchDeleteSubmissions(submissionIDs []string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("submission_id IN ?", submissionIDs).Delete(&model.PostClassTestAnswer{}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", submissionIDs).Delete(&model.PostClassTestSubmission{}).Error
	})
}

func (r *PostClassTestRepository) FindSubmissionByUserAndTest(userID uint, testID string) (*model.PostClassTestSubmission, error) {
	var s model.PostClassTestSubmission
	err := r.DB.Where("user_id = ? AND test_id = ?", userID, testID).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostClassTestRepository) UnpublishAllExcept(testID string) error {
	return r.DB.Model(&model.PostClassTest{}).
		Where("id <> ? AND is_published = ?", testID, true).
		Update("is_published", false).Error
}
