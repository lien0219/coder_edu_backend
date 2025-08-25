package repository

import (
	"coder_edu_backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type TaskRepository struct {
	DB *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{DB: db}
}

func (r *TaskRepository) Create(task *model.Task) error {
	return r.DB.Create(task).Error
}

func (r *TaskRepository) FindByID(id uint) (*model.Task, error) {
	var task model.Task
	err := r.DB.First(&task, id).Error
	return &task, err
}

func (r *TaskRepository) FindByUserID(userID uint) ([]*model.Task, error) {
	var tasks []*model.Task
	err := r.DB.Where("user_id = ?", userID).Find(&tasks).Error
	return tasks, err
}

func (r *TaskRepository) Update(task *model.Task) error {
	return r.DB.Save(task).Error
}

func (r *TaskRepository) FindTodayTasks(userID uint) ([]*model.Task, error) {
	var tasks []*model.Task
	err := r.DB.Where("user_id = ? AND due_date >= CURDATE() AND due_date < CURDATE() + INTERVAL 1 DAY", userID).Find(&tasks).Error
	return tasks, err
}
func (r *TaskRepository) FindByUserAndDate(userID uint, date time.Time) ([]*model.Task, error) {
	var tasks []*model.Task
	err := r.DB.Where("user_id = ? AND due_date >= ? AND due_date < ?",
		userID, date.Format("2006-01-02"), date.AddDate(0, 0, 1).Format("2006-01-02")).
		Find(&tasks).Error
	return tasks, err
}

func (r *TaskRepository) UpdateStatus(id uint, status model.TaskStatus) error {
	return r.DB.Model(&model.Task{}).
		Where("id = ?", id).
		Update("status", status).
		Error
}
func (r *TaskRepository) FindByModuleTypeAndUser(moduleType string, userID uint) ([]*model.Task, error) {
	var tasks []*model.Task
	err := r.DB.Where("user_id = ? AND module_type = ?", userID, moduleType).Find(&tasks).Error
	return tasks, err
}

func (r *TaskRepository) FindTransferTasks(userID uint) ([]*model.Task, error) {
	var tasks []*model.Task
	err := r.DB.Where("user_id = ? AND is_transfer_task = ?", userID, true).Find(&tasks).Error
	return tasks, err
}
