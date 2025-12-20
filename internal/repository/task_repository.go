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

// CreateWeeklyTask 创建周任务
func (r *TaskRepository) CreateWeeklyTask(task *model.TeacherWeeklyTask) error {
	if err := r.DB.Create(task).Error; err != nil {
		return err
	}

	// 在事务外批量插入任务项，减少锁竞争
	db, err := r.DB.DB()
	if err != nil {
		return err
	}

	// 准备一个批量插入的预处理语句
	stmt, err := db.Prepare("INSERT INTO task_items (weekly_task_id, day_of_week, item_type, resource_id, exercise_id, title, description, content_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close() // 使用defer确保无论如何都会关闭

	// 开始事务进行批量插入
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// 使用新事务的stmt
	txStmt, err := tx.Prepare("INSERT INTO task_items (weekly_task_id, day_of_week, item_type, resource_id, exercise_id, title, description, content_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer txStmt.Close()

	// 批量执行插入
	for i := range task.TaskItems {
		item := &task.TaskItems[i]
		var (
			taskID    = task.ID
			itemDay   = string(item.DayOfWeek)
			itemType  = string(item.ItemType)
			resID     = item.ResourceID
			exerID    = item.ExerciseID
			itemTitle = item.Title
			itemDesc  = item.Description
			itemCType = item.ContentType
		)

		_, err = txStmt.Exec(taskID, itemDay, itemType, resID, exerID, itemTitle, itemDesc, itemCType)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// GetWeeklyTaskByTeacherAndDate 根据老师ID、资源分类ID和日期获取周任务
func (r *TaskRepository) GetWeeklyTaskByTeacherAndDate(teacherID uint, resourceModuleID uint, date time.Time) (*model.TeacherWeeklyTask, error) {
	var task model.TeacherWeeklyTask
	// 计算本周的开始和结束日期
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := date.AddDate(0, 0, -(weekday - 1))
	weekEnd := weekStart.AddDate(0, 0, 6)

	query := r.DB.Preload("TaskItems").Where("teacher_id = ? AND week_start_date = ? AND week_end_date = ?",
		teacherID, weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02"))

	// 如果提供了资源分类ID，则添加到查询条件中
	if resourceModuleID > 0 {
		query = query.Where("resource_module_id = ?", resourceModuleID)
	}

	err := query.First(&task).Error
	return &task, err
}

// UpdateWeeklyTask 更新周任务
func (r *TaskRepository) UpdateWeeklyTask(task *model.TeacherWeeklyTask) error {
	// 首先保存周任务本身
	if err := r.DB.Save(task).Error; err != nil {
		return err
	}

	db, err := r.DB.DB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec("DELETE FROM task_items WHERE weekly_task_id = ?", task.ID); err != nil {
		tx.Rollback()
		return err
	}

	txStmt, err := tx.Prepare("INSERT INTO task_items (weekly_task_id, day_of_week, item_type, resource_id, exercise_id, title, description, content_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer txStmt.Close()

	for i := range task.TaskItems {
		item := &task.TaskItems[i]
		var (
			taskID    = task.ID
			itemDay   = string(item.DayOfWeek)
			itemType  = string(item.ItemType)
			resID     = item.ResourceID
			exerID    = item.ExerciseID
			itemTitle = item.Title
			itemDesc  = item.Description
			itemCType = item.ContentType
		)

		_, err = txStmt.Exec(taskID, itemDay, itemType, resID, exerID, itemTitle, itemDesc, itemCType)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := r.DB.Where("weekly_task_id = ?", task.ID).Find(&task.TaskItems).Error; err != nil {
		return nil
	}

	return nil
}

// CreateDailyTaskCompletion 创建每日任务完成记录
func (r *TaskRepository) CreateDailyTaskCompletion(completion *model.DailyTaskCompletion) error {
	return r.DB.Create(completion).Error
}

// GetDailyTaskCompletion 获取每日任务完成记录
func (r *TaskRepository) GetDailyTaskCompletion(userID, taskItemID uint, date time.Time) (*model.DailyTaskCompletion, error) {
	var completion model.DailyTaskCompletion
	err := r.DB.Where("user_id = ? AND task_item_id = ? AND DATE(completion_date) = DATE(?)",
		userID, taskItemID, date).First(&completion).Error
	return &completion, err
}

// UpdateDailyTaskCompletion 更新每日任务完成记录
func (r *TaskRepository) UpdateDailyTaskCompletion(completion *model.DailyTaskCompletion) error {
	return r.DB.Save(completion).Error
}

// GetTodayTasks 获取今天的任务列表
func (r *TaskRepository) GetTodayTasks(resourceModuleID uint, dayOfWeek model.Weekday) ([]model.TaskItem, error) {
	today := time.Now()
	weekday := int(today.Weekday())
	// 计算周一的日期：如果今天是周日(0)，则减去6天；否则减去(weekday-1)天
	var weekStart time.Time
	if weekday == 0 {
		weekStart = time.Date(today.Year(), today.Month(), today.Day()-6, 0, 0, 0, 0, today.Location())
	} else {
		weekStart = time.Date(today.Year(), today.Month(), today.Day()-(weekday-1), 0, 0, 0, 0, today.Location())
	}
	weekEnd := weekStart.AddDate(0, 0, 6) // 周日

	var taskItems []model.TaskItem

	// 查询当前周里所有模块中当天的任务（不再按 resourceModuleID 精确匹配）
	query := r.DB.Joins("JOIN teacher_weekly_tasks ON task_items.weekly_task_id = teacher_weekly_tasks.id").
		Where("task_items.day_of_week = ? AND teacher_weekly_tasks.week_start_date = ? AND teacher_weekly_tasks.week_end_date = ?",
			dayOfWeek, weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02"))

	err := query.Find(&taskItems).Error
	return taskItems, err
}

// GetAllTodayTasks 获取所有资源模块的今天任务列表
func (r *TaskRepository) GetAllTodayTasks(dayOfWeek model.Weekday) ([]model.TaskItem, error) {
	today := time.Now()
	weekday := int(today.Weekday())
	// 计算周一的日期：如果今天是周日(0)，则减去6天；否则减去(weekday-1)天
	var weekStart time.Time
	if weekday == 0 {
		weekStart = time.Date(today.Year(), today.Month(), today.Day()-6, 0, 0, 0, 0, today.Location())
	} else {
		weekStart = time.Date(today.Year(), today.Month(), today.Day()-(weekday-1), 0, 0, 0, 0, today.Location())
	}
	weekEnd := weekStart.AddDate(0, 0, 6) // 周日

	var taskItems []model.TaskItem
	query := r.DB.Joins("JOIN teacher_weekly_tasks ON task_items.weekly_task_id = teacher_weekly_tasks.id").
		Where("task_items.day_of_week = ? AND teacher_weekly_tasks.week_start_date = ? AND teacher_weekly_tasks.week_end_date = ?",
			dayOfWeek, weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02"))

	err := query.Find(&taskItems).Error
	return taskItems, err
}

// CheckTaskItemExists 检查任务项是否已存在
func (r *TaskRepository) CheckTaskItemExists(weeklyTaskID uint, dayOfWeek model.Weekday, itemType model.TaskItemType, resourceID, exerciseID uint) (bool, error) {
	var count int64
	query := r.DB.Model(&model.TaskItem{}).Where("weekly_task_id = ? AND day_of_week = ? AND item_type = ?", weeklyTaskID, dayOfWeek, itemType)

	if itemType == model.TaskItemExercise && exerciseID > 0 {
		query = query.Where("exercise_id = ?", exerciseID)
	} else if resourceID > 0 {
		query = query.Where("resource_id = ?", resourceID)
	}

	err := query.Count(&count).Error
	return count > 0, err
}

// GetWeeklyTasksWithPagination 获取老师的周任务列表，支持分页和搜索
func (r *TaskRepository) GetWeeklyTasksWithPagination(teacherID uint, page, limit int, search string) ([]model.TeacherWeeklyTask, int, error) {
	var tasks []model.TeacherWeeklyTask
	var total int64

	query := r.DB.Model(&model.TeacherWeeklyTask{}).Where("teacher_id = ?", teacherID)

	// 搜索功能
	if search != "" {
		query = query.Where("resource_module_name LIKE ?", "%"+search+"%")
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * limit
	err := query.Preload("TaskItems").
		Order("week_start_date DESC").
		Offset(offset).Limit(limit).
		Find(&tasks).Error

	return tasks, int(total), err
}

// DeleteWeeklyTask 删除周任务
func (r *TaskRepository) DeleteWeeklyTask(taskID uint, teacherID uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// 先删除关联的任务项
		if err := tx.Where("weekly_task_id = ?", taskID).Delete(&model.TaskItem{}).Error; err != nil {
			return err
		}

		// 删除周任务记录
		result := tx.Where("id = ? AND teacher_id = ?", taskID, teacherID).Delete(&model.TeacherWeeklyTask{})
		if result.Error != nil {
			return result.Error
		}

		// 检查是否找到并删除了记录
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}
