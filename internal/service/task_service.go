package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TaskService 处理任务相关的业务逻辑
type TaskService struct {
	TaskRepo           *repository.TaskRepository
	ResourceRepo       *repository.ResourceRepository
	ExerciseRepo       *repository.ExerciseQuestionRepository
	ResourceModuleRepo *repository.CProgrammingResourceRepository
	GoalRepo           *repository.GoalRepository
}

func NewTaskService(
	taskRepo *repository.TaskRepository,
	resourceRepo *repository.ResourceRepository,
	exerciseRepo *repository.ExerciseQuestionRepository,
	resourceModuleRepo *repository.CProgrammingResourceRepository,
	goalRepo *repository.GoalRepository,
) *TaskService {
	return &TaskService{
		TaskRepo:           taskRepo,
		ResourceRepo:       resourceRepo,
		ExerciseRepo:       exerciseRepo,
		ResourceModuleRepo: resourceModuleRepo,
		GoalRepo:           goalRepo,
	}
}

// SetWeeklyTask 设置周任务
func (s *TaskService) SetWeeklyTask(teacherID, resourceModuleID uint, taskItems []model.TaskItem) (*model.TeacherWeeklyTask, error) {
	// 获取资源模块信息
	resourceModule, err := s.ResourceModuleRepo.FindByID(resourceModuleID)
	if err != nil {
		return nil, fmt.Errorf("资源模块不存在 (ID: %d)", resourceModuleID)
	}

	// 计算本周的开始和结束日期
	today := time.Now()
	weekday := int(today.Weekday())
	if weekday == 0 { // Sunday
		weekday = 7
	}
	weekStart := time.Date(today.Year(), today.Month(), today.Day()-weekday+1, 0, 0, 0, 0, today.Location())
	weekEnd := weekStart.AddDate(0, 0, 6)

	// 检查是否已有本周任务
	existingTask, err := s.TaskRepo.GetWeeklyTaskByTeacherAndDate(teacherID, resourceModuleID, today)
	var weeklyTask *model.TeacherWeeklyTask

	if err == nil {
		// 更新现有任务
		weeklyTask = existingTask
		// 删除旧的任务项（在验证新任务项之前删除，避免重复检查）
		if err := s.TaskRepo.DB.Where("weekly_task_id = ?", weeklyTask.ID).Delete(&model.TaskItem{}).Error; err != nil {
			return nil, fmt.Errorf("删除旧任务项失败: %v", err)
		}
		weeklyTask.TaskItems = taskItems
	} else {
		// 创建新任务
		weeklyTask = &model.TeacherWeeklyTask{
			TeacherID:          teacherID,
			ResourceModuleID:   resourceModuleID,
			ResourceModuleName: resourceModule.Name,
			WeekStartDate:      weekStart,
			WeekEndDate:        weekEnd,
			TaskItems:          taskItems,
		}
	}

	// 验证并完善任务项信息
	for i := range taskItems {
		item := &taskItems[i]
		item.WeeklyTaskID = weeklyTask.ID
		item.ID = 0

		// 根据类型获取资源信息
		if item.ItemType == model.TaskItemVideo || item.ItemType == model.TaskItemArticle {
			resource, err := s.ResourceRepo.FindByID(item.ResourceID)
			if err != nil {
				return nil, fmt.Errorf("资源不存在 (ID: %d)", item.ResourceID)
			}
			item.Title = resource.Title
			item.Description = resource.Description
			item.ContentType = string(resource.Type)
		} else if item.ItemType == model.TaskItemExercise {
			exercise, err := s.ExerciseRepo.FindByID(item.ExerciseID)
			if err != nil {
				return nil, fmt.Errorf("练习题不存在 (ID: %d)", item.ExerciseID)
			}
			item.Title = exercise.Title
			item.Description = exercise.Description
			item.ContentType = "exercise"
		}
	}

	// 保存周任务
	if err == nil {
		if err := s.TaskRepo.UpdateWeeklyTask(weeklyTask); err != nil {
			return nil, fmt.Errorf("更新周任务失败: %v", err)
		}
	} else {
		if err := s.TaskRepo.CreateWeeklyTask(weeklyTask); err != nil {
			return nil, fmt.Errorf("创建周任务失败: %v", err)
		}
	}

	return weeklyTask, nil
}

// GetTodayTasks 获取今天的任务列表
func (s *TaskService) GetTodayTasks(userID, resourceModuleID uint) ([]map[string]interface{}, error) {
	// 获取今天是星期几
	var dayOfWeek model.Weekday
	todayWeekday := time.Now().Weekday()
	switch todayWeekday {
	case time.Monday:
		dayOfWeek = model.Monday
	case time.Tuesday:
		dayOfWeek = model.Tuesday
	case time.Wednesday:
		dayOfWeek = model.Wednesday
	case time.Thursday:
		dayOfWeek = model.Thursday
	case time.Friday:
		dayOfWeek = model.Friday
	case time.Saturday:
		dayOfWeek = model.Saturday
	case time.Sunday:
		dayOfWeek = model.Sunday
	}

	// 获取今天的任务项（仅查询当前周；不回退到历史周）
	var taskItems []model.TaskItem
	var err error
	if resourceModuleID == 0 {
		taskItems, err = s.TaskRepo.GetAllTodayTasks(dayOfWeek)
	} else {
		taskItems, err = s.TaskRepo.GetTodayTasks(resourceModuleID, dayOfWeek)
	}
	if err != nil {
		return nil, err
	}

	return s.buildTaskResult(taskItems, userID), nil
}

// buildTaskResult 构建任务结果列表
func (s *TaskService) buildTaskResult(taskItems []model.TaskItem, userID uint) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(taskItems))
	for _, item := range taskItems {
		// 获取对应的周任务信息以获取资源模块ID
		var weeklyTask model.TeacherWeeklyTask
		resourceModuleID := uint(0)
		if err := s.TaskRepo.DB.First(&weeklyTask, item.WeeklyTaskID).Error; err == nil {
			resourceModuleID = weeklyTask.ResourceModuleID
		}

		// 构建任务项信息
		taskInfo := map[string]interface{}{
			"id":                item.ID,
			"dayOfWeek":         item.DayOfWeek,
			"itemType":          item.ItemType,
			"resourceId":        item.ResourceID,
			"exerciseId":        item.ExerciseID,
			"title":             item.Title,
			"description":       item.Description,
			"contentType":       item.ContentType,
			"resourceModuleId":  resourceModuleID,
			"isCompleted":       false,
			"progress":          0.0,
			"resourceCompleted": false,
		}

		// 获取任务完成状态
		completion, err := s.TaskRepo.GetDailyTaskCompletion(userID, item.ID, time.Now())
		if err == nil {
			taskInfo["isCompleted"] = completion.IsCompleted
			taskInfo["progress"] = completion.Progress
			taskInfo["resourceCompleted"] = completion.ResourceCompleted
		}

		result = append(result, taskInfo)
	}

	return result
}

// UpdateTaskCompletion 更新任务完成状态
func (s *TaskService) UpdateTaskCompletion(userID, taskItemID uint, isCompleted bool, progress float64, resourceCompleted bool) error {
	// 获取任务项信息
	var taskItem model.TaskItem
	if err := s.TaskRepo.DB.First(&taskItem, taskItemID).Error; err != nil {
		return errors.New("任务项不存在")
	}

	// 检查是否已有完成记录
	completion, err := s.TaskRepo.GetDailyTaskCompletion(userID, taskItemID, time.Now())
	if err != nil {
		// 创建新的完成记录
		completion = &model.DailyTaskCompletion{
			UserID:         userID,
			TaskItemID:     taskItemID,
			CompletionDate: time.Now(),
		}
	}

	// 更新完成状态
	completion.IsCompleted = isCompleted
	completion.Progress = progress
	completion.ResourceCompleted = resourceCompleted

	if err := s.TaskRepo.UpdateDailyTaskCompletion(completion); err != nil {
		return err
	}

	return nil
}

// GetWeeklyTasks 获取老师的周任务列表
func (s *TaskService) GetWeeklyTasks(teacherID uint, page, limit int, search string) ([]model.TeacherWeeklyTask, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	return s.TaskRepo.GetWeeklyTasksWithPagination(teacherID, page, limit, search)
}

// GetCurrentWeekTask 获取当前周任务
func (s *TaskService) GetCurrentWeekTask(teacherID uint, resourceModuleID uint) (*model.TeacherWeeklyTask, error) {
	// 如果指定了资源模块ID，只获取该模块的任务
	if resourceModuleID > 0 {
		task, err := s.TaskRepo.GetWeeklyTaskByTeacherAndDate(teacherID, resourceModuleID, time.Now())
		if err != nil {
			return nil, err
		}

		// 确保返回的任务包含正确的资源模块名称
		resourceModule, err := s.ResourceModuleRepo.FindByID(task.ResourceModuleID)
		if err == nil && resourceModule != nil {
			task.ResourceModuleName = resourceModule.Name
		}

		return task, nil
	}

	// 如果没有指定资源模块ID，获取老师本周的所有任务
	// 获取本周的日期范围（周一到周日）
	now := time.Now()
	weekday := int(now.Weekday())
	// 计算周一的日期：如果今天是周日(0)，则减去6天；否则减去(weekday-1)天
	var weekStart time.Time
	if weekday == 0 {
		weekStart = time.Date(now.Year(), now.Month(), now.Day()-6, 0, 0, 0, 0, now.Location())
	} else {
		weekStart = time.Date(now.Year(), now.Month(), now.Day()-(weekday-1), 0, 0, 0, 0, now.Location())
	}
	weekEnd := weekStart.AddDate(0, 0, 6) // 周日

	// 获取本周的所有周任务 - 修改查询逻辑，获取老师所有的周任务
	var allTasks []model.TeacherWeeklyTask

	// 首先尝试严格匹配本周
	err := s.TaskRepo.DB.Where("teacher_id = ? AND week_start_date = ? AND week_end_date = ?",
		teacherID, weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02")).
		Preload("TaskItems").Find(&allTasks).Error

	if err != nil {
		return nil, err
	}

	// 如果本周任务不完整（任务项总数较少），获取老师所有的周任务
	totalTaskItems := 0
	for _, task := range allTasks {
		totalTaskItems += len(task.TaskItems)
	}

	if totalTaskItems < 10 { // 如果任务项总数少于10个，获取所有相关任务
		var allTeacherTasks []model.TeacherWeeklyTask
		err = s.TaskRepo.DB.Where("teacher_id = ?", teacherID).
			Order("week_start_date DESC").
			Limit(10). // 限制获取最近10个周任务
			Preload("TaskItems").Find(&allTeacherTasks).Error

		if err == nil {
			allTasks = allTeacherTasks // 使用所有任务替代本周任务
		}
	}

	if err != nil {
		return nil, err
	}

	if len(allTasks) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	// 合并所有任务项（保持原始顺序）
	allMergedItems := make([]model.TaskItem, 0)
	for _, task := range allTasks {
		allMergedItems = append(allMergedItems, task.TaskItems...)
	}

	// 返回一个虚拟的周任务，包含所有合并后的任务项
	result := &model.TeacherWeeklyTask{
		TeacherID:     teacherID,
		WeekStartDate: weekStart,
		WeekEndDate:   weekEnd,
		TaskItems:     allMergedItems,
	}

	return result, nil
}

// DeleteWeeklyTask 删除周任务
func (s *TaskService) DeleteWeeklyTask(taskID uint, teacherID uint) error {
	return s.TaskRepo.DeleteWeeklyTask(taskID, teacherID)
}
