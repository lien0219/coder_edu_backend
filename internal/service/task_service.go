package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"errors"
	"time"
)

// TaskService 处理任务相关的业务逻辑
type TaskService struct {
	TaskRepo           *repository.TaskRepository
	ResourceRepo       *repository.ResourceRepository
	ExerciseRepo       *repository.ExerciseQuestionRepository
	ResourceModuleRepo *repository.CProgrammingResourceRepository
}

func NewTaskService(
	taskRepo *repository.TaskRepository,
	resourceRepo *repository.ResourceRepository,
	exerciseRepo *repository.ExerciseQuestionRepository,
	resourceModuleRepo *repository.CProgrammingResourceRepository,
) *TaskService {
	return &TaskService{
		TaskRepo:           taskRepo,
		ResourceRepo:       resourceRepo,
		ExerciseRepo:       exerciseRepo,
		ResourceModuleRepo: resourceModuleRepo,
	}
}

// SetWeeklyTask 设置周任务
func (s *TaskService) SetWeeklyTask(teacherID, resourceModuleID uint, taskItems []model.TaskItem) (*model.TeacherWeeklyTask, error) {
	// 获取资源模块信息
	resourceModule, err := s.ResourceModuleRepo.FindByID(resourceModuleID)
	if err != nil {
		return nil, errors.New("资源模块不存在")
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
	existingTask, err := s.TaskRepo.GetWeeklyTaskByTeacherAndDate(teacherID, today)
	var weeklyTask *model.TeacherWeeklyTask

	if err == nil {
		// 更新现有任务
		weeklyTask = existingTask
		// 删除旧的任务项
		for _, item := range weeklyTask.TaskItems {
			s.TaskRepo.DB.Delete(&item)
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

		// 检查任务项是否已存在
		exists, err := s.TaskRepo.CheckTaskItemExists(weeklyTask.ID, item.DayOfWeek, item.ItemType,
			item.ResourceID, item.ExerciseID)
		if err == nil && exists {
			return nil, errors.New("同一资源分类下不能选择相同的内容作为同一任务")
		}

		// 根据类型获取资源信息
		if item.ItemType == model.TaskItemVideo || item.ItemType == model.TaskItemArticle {
			resource, err := s.ResourceRepo.FindByID(item.ResourceID)
			if err != nil {
				return nil, errors.New("资源不存在")
			}
			item.Title = resource.Title
			item.Description = resource.Description
			item.ContentType = string(resource.Type)
		} else if item.ItemType == model.TaskItemExercise {
			exercise, err := s.ExerciseRepo.FindByID(item.ExerciseID)
			if err != nil {
				return nil, errors.New("练习题不存在")
			}
			item.Title = exercise.Title
			item.Description = exercise.Description
			item.ContentType = "exercise"
		}
	}

	// 保存周任务
	if err == nil {
		if err := s.TaskRepo.UpdateWeeklyTask(weeklyTask); err != nil {
			return nil, err
		}
	} else {
		if err := s.TaskRepo.CreateWeeklyTask(weeklyTask); err != nil {
			return nil, err
		}
	}

	return weeklyTask, nil
}

// GetTodayTasks 获取今天的任务列表
func (s *TaskService) GetTodayTasks(userID, resourceModuleID uint) ([]map[string]interface{}, error) {
	// 获取今天是星期几
	var dayOfWeek model.Weekday
	switch time.Now().Weekday() {
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

	// 获取今天的任务项
	taskItems, err := s.TaskRepo.GetTodayTasks(resourceModuleID, dayOfWeek)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(taskItems))
	for _, item := range taskItems {
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

	return result, nil
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

// DeleteWeeklyTask 删除周任务
func (s *TaskService) DeleteWeeklyTask(taskID uint, teacherID uint) error {
	return s.TaskRepo.DeleteWeeklyTask(taskID, teacherID)
}
