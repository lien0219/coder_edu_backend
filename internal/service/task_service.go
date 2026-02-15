package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"coder_edu_backend/internal/util"
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
	seen := make(map[string]bool)
	for i := range taskItems {
		item := &taskItems[i]

		// 检查同一天是否添加了相同的资源
		key := fmt.Sprintf("%s-%s-%d-%d", item.DayOfWeek, item.ItemType, item.ResourceID, item.ExerciseID)
		if seen[key] {
			return nil, fmt.Errorf("同一天不能添加重复的任务项")
		}
		seen[key] = true

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
	if len(taskItems) == 0 {
		return []map[string]interface{}{}
	}

	// 批量获取任务完成状态
	taskItemIDs := make([]uint, len(taskItems))
	for i, item := range taskItems {
		taskItemIDs[i] = item.ID
	}

	completions, _ := s.TaskRepo.GetDailyTaskCompletionsByTaskItemIDs(userID, taskItemIDs, time.Now())
	completionMap := make(map[uint]model.DailyTaskCompletion)
	for _, c := range completions {
		completionMap[c.TaskItemID] = c
	}

	result := make([]map[string]interface{}, 0, len(taskItems))
	for _, item := range taskItems {
		// 获取对应的周任务信息以获取资源模块ID
		resourceModuleID := uint(0)
		if item.WeeklyTask != nil {
			resourceModuleID = item.WeeklyTask.ResourceModuleID
		} else if weeklyTask, err := s.TaskRepo.FindWeeklyTaskByID(item.WeeklyTaskID); err == nil {
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
		if completion, ok := completionMap[item.ID]; ok {
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
	if _, err := s.TaskRepo.FindTaskItemByID(taskItemID); err != nil {
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
func (s *TaskService) GetCurrentWeekTask(userID uint, role model.UserRole, resourceModuleID uint, targetDate time.Time) (interface{}, error) {
	if targetDate.IsZero() {
		targetDate = time.Now()
	}

	// 计算目标日期所在的周一和周日
	weekday := int(targetDate.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day()-(weekday-1), 0, 0, 0, 0, targetDate.Location())
	weekEnd := weekStart.AddDate(0, 0, 6)

	weekStartStr := weekStart.Format(util.DateFormat)
	weekEndStr := weekEnd.Format(util.DateFormat)

	// 如果指定了资源模块ID，获取该模块的任务
	if resourceModuleID > 0 {
		var teacherID uint
		if role != model.Student {
			teacherID = userID
		}

		task, err := s.TaskRepo.FindWeeklyTaskByModuleAndWeek(resourceModuleID, weekStartStr, weekEndStr, teacherID)
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

	// 如果没有指定资源模块ID，获取本周的所有任务
	var teacherID uint
	if role != model.Student {
		teacherID = userID
	}

	allTasks, err := s.TaskRepo.FindWeeklyTasksByWeek(weekStartStr, weekEndStr, teacherID)
	if err != nil {
		return nil, err
	}

	if len(allTasks) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	// 构建分组返回结果
	groups := make([]map[string]interface{}, 0)
	totalTaskItems := 0

	for _, task := range allTasks {
		// 按星期几分组
		dayGroups := make(map[string][]model.TaskItem)
		for _, item := range task.TaskItems {
			dayGroups[string(item.DayOfWeek)] = append(dayGroups[string(item.DayOfWeek)], item)
		}

		groups = append(groups, map[string]interface{}{
			"resourceModuleId":   task.ResourceModuleID,
			"resourceModuleName": task.ResourceModuleName,
			"taskItems":          task.TaskItems,
			"dayGroups":          dayGroups,
		})
		totalTaskItems += len(task.TaskItems)
	}

	return map[string]interface{}{
		"weekStartDate":  weekStart,
		"weekEndDate":    weekEnd,
		"groups":         groups,
		"totalTaskItems": totalTaskItems,
	}, nil
}

// DeleteWeeklyTask 删除周任务
func (s *TaskService) DeleteWeeklyTask(taskID uint, teacherID uint) error {
	return s.TaskRepo.DeleteWeeklyTask(taskID, teacherID)
}
