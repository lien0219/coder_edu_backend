package repository

import (
	"coder_edu_backend/internal/model"

	"gorm.io/gorm"
)

type AssessmentRepository struct {
	DB *gorm.DB
}

func NewAssessmentRepository(db *gorm.DB) *AssessmentRepository {
	return &AssessmentRepository{DB: db}
}

func (r *AssessmentRepository) CreateQuestion(question *model.AssessmentQuestion) error {
	return r.DB.Create(question).Error
}

func (r *AssessmentRepository) FindQuestionByID(id uint) (*model.AssessmentQuestion, error) {
	var q model.AssessmentQuestion
	err := r.DB.First(&q, id).Error
	return &q, err
}

func (r *AssessmentRepository) ListQuestions(assessmentID uint, page, limit int) ([]model.AssessmentQuestion, int64, error) {
	var qs []model.AssessmentQuestion
	var total int64
	query := r.DB.Model(&model.AssessmentQuestion{})
	if assessmentID > 0 {
		query = query.Where("assessment_id = ?", assessmentID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * limit
	err := query.Order("`order` asc, created_at desc").Offset(offset).Limit(limit).Find(&qs).Error
	return qs, total, err
}

func (r *AssessmentRepository) ListAllQuestions(assessmentID uint) ([]model.AssessmentQuestion, error) {
	var qs []model.AssessmentQuestion
	query := r.DB.Model(&model.AssessmentQuestion{})
	if assessmentID > 0 {
		query = query.Where("assessment_id = ?", assessmentID)
	}
	err := query.Order("`order` asc, created_at desc").Find(&qs).Error
	return qs, err
}

func (r *AssessmentRepository) UpdateQuestion(question *model.AssessmentQuestion) error {
	return r.DB.Save(question).Error
}

func (r *AssessmentRepository) DeleteQuestion(id uint) error {
	return r.DB.Delete(&model.AssessmentQuestion{}, id).Error
}

// Assessment related methods
func (r *AssessmentRepository) CreateAssessment(a *model.Assessment) error {
	return r.DB.Create(a).Error
}

func (r *AssessmentRepository) FindAssessmentByID(id uint) (*model.Assessment, error) {
	var a model.Assessment
	err := r.DB.First(&a, id).Error
	return &a, err
}

func (r *AssessmentRepository) ListAssessments(page, limit int) ([]model.Assessment, int64, error) {
	var as []model.Assessment
	var total int64
	query := r.DB.Model(&model.Assessment{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * limit
	err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&as).Error
	return as, total, err
}

func (r *AssessmentRepository) CreateSubmission(s *model.AssessmentSubmission) error {
	return r.DB.Create(s).Error
}

func (r *AssessmentRepository) ListSubmissions(page, limit int, status string, studentName string) ([]model.AssessmentSubmission, int64, error) {
	var ss []model.AssessmentSubmission
	var total int64

	query := r.DB.Table("users").
		Select("assessment_submissions.*, users.id as user_id_from_user, users.name as user_name, users.email as user_email").
		Joins("LEFT JOIN assessment_submissions ON users.id = assessment_submissions.user_id AND assessment_submissions.deleted_at IS NULL").
		Where("users.role = ?", "student").
		Where("users.deleted_at IS NULL")

	if status != "" {
		if status == "untested" {
			query = query.Where("assessment_submissions.id IS NULL")
		} else {
			query = query.Where("assessment_submissions.status = ?", status)
		}
	}

	if studentName != "" {
		query = query.Where("users.name LIKE ?", "%"+studentName+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit

	type resultRow struct {
		model.AssessmentSubmission
		UserIDFromUser uint   `gorm:"column:user_id_from_user"`
		UserName       string `gorm:"column:user_name"`
		UserEmail      string `gorm:"column:user_email"`
	}
	var rows []resultRow

	err := query.Order("assessment_submissions.created_at desc, users.created_at desc").
		Offset(offset).Limit(limit).
		Scan(&rows).Error

	if err != nil {
		return nil, 0, err
	}

	ss = make([]model.AssessmentSubmission, len(rows))
	for i, row := range rows {
		ss[i] = row.AssessmentSubmission
		if ss[i].ID == 0 {
			ss[i].Status = "untested"
			ss[i].UserID = row.UserIDFromUser
		}
		ss[i].User = &model.User{
			Name:  row.UserName,
			Email: row.UserEmail,
		}
		ss[i].User.ID = row.UserIDFromUser
	}

	return ss, total, err
}

func (r *AssessmentRepository) FindSubmissionByID(id uint) (*model.AssessmentSubmission, error) {
	var s model.AssessmentSubmission
	err := r.DB.Preload("User").First(&s, id).Error
	return &s, err
}

func (r *AssessmentRepository) UpdateSubmission(s *model.AssessmentSubmission) error {
	return r.DB.Save(s).Error
}

func (r *AssessmentRepository) FindSubmissionByUserAndAssessment(userID, assessmentID uint) (*model.AssessmentSubmission, error) {
	var s model.AssessmentSubmission
	err := r.DB.Where("user_id = ? AND assessment_id = ?", userID, assessmentID).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *AssessmentRepository) DeleteSubmission(id uint) error {
	return r.DB.Delete(&model.AssessmentSubmission{}, id).Error
}

func (r *AssessmentRepository) UpdateUserAssessmentStatus(userID uint, canTake bool) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Update("can_take_assessment", canTake).Error
}

func (r *AssessmentRepository) BatchUpdateUserAssessmentStatus(userIDs []uint, canTake bool) error {
	return r.DB.Model(&model.User{}).Where("id IN ?", userIDs).Update("can_take_assessment", canTake).Error
}

func (r *AssessmentRepository) GetUserAssessmentStatus(userID uint) (bool, error) {
	var user model.User
	err := r.DB.Select("can_take_assessment").First(&user, userID).Error
	return user.CanTakeAssessment, err
}
