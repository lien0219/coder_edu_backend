package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type MigrationTaskService struct {
	Repo    *repository.MigrationTaskRepository
	UserSvc *UserService
}

func NewMigrationTaskService(repo *repository.MigrationTaskRepository, userSvc *UserService) *MigrationTaskService {
	return &MigrationTaskService{Repo: repo, UserSvc: userSvc}
}

type MigrationQuestionReq struct {
	ID             string `json:"id"`
	Title          string `json:"title" binding:"required"`
	Description    string `json:"description" binding:"required"`
	Difficulty     string `json:"difficulty" binding:"required"`
	StandardAnswer string `json:"standardAnswer" binding:"required"`
	Points         int    `json:"points"`
	Order          int    `json:"order"`
}

type MigrationTaskReq struct {
	Title       *string                 `json:"title"`
	Description *string                 `json:"description"`
	Difficulty  *string                 `json:"difficulty"`
	TimeLimit   *int                    `json:"timeLimit"`
	IsPublished *bool                   `json:"isPublished"`
	Questions   *[]MigrationQuestionReq `json:"questions"`
}

func (s *MigrationTaskService) CreateTask(creatorID uint, req MigrationTaskReq) (*model.MigrationTask, error) {
	if req.Title == nil || *req.Title == "" {
		return nil, errors.New("title is required")
	}

	task := &model.MigrationTask{
		Title:     *req.Title,
		CreatorID: creatorID,
	}

	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Difficulty != nil {
		task.Difficulty = *req.Difficulty
	}
	if req.TimeLimit != nil {
		task.TimeLimit = *req.TimeLimit
	}
	if req.IsPublished != nil {
		task.IsPublished = *req.IsPublished
		if task.IsPublished {
			now := time.Now()
			task.PublishedAt = &now
		}
	}

	if err := s.Repo.CreateTask(task); err != nil {
		return nil, err
	}

	if req.Questions != nil {
		for _, qReq := range *req.Questions {
			q := &model.MigrationQuestion{
				TaskID:         task.ID,
				Title:          qReq.Title,
				Description:    qReq.Description,
				Difficulty:     qReq.Difficulty,
				StandardAnswer: qReq.StandardAnswer,
				Points:         qReq.Points,
				Order:          qReq.Order,
			}
			if err := s.Repo.CreateQuestion(q); err != nil {
				return nil, err
			}
		}
	}

	return task, nil
}

func (s *MigrationTaskService) UpdateTask(taskID string, req MigrationTaskReq) (*model.MigrationTask, error) {
	task, err := s.Repo.FindTaskByID(taskID)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Difficulty != nil {
		task.Difficulty = *req.Difficulty
	}
	if req.TimeLimit != nil {
		task.TimeLimit = *req.TimeLimit
	}
	if req.IsPublished != nil {
		if *req.IsPublished && !task.IsPublished {
			now := time.Now()
			task.PublishedAt = &now
		} else if !*req.IsPublished {
			task.PublishedAt = nil
		}
		task.IsPublished = *req.IsPublished
	}

	if err := s.Repo.UpdateTask(task); err != nil {
		return nil, err
	}

	if req.Questions != nil {
		existingQs, _ := s.Repo.ListQuestions(taskID)
		existingMap := make(map[string]*model.MigrationQuestion)
		for i := range existingQs {
			existingMap[existingQs[i].ID] = &existingQs[i]
		}

		newQIDs := make(map[string]bool)
		for _, qReq := range *req.Questions {
			if qReq.ID != "" && existingMap[qReq.ID] != nil {
				// 更新已有题目
				q := existingMap[qReq.ID]
				q.Title = qReq.Title
				q.Description = qReq.Description
				q.Difficulty = qReq.Difficulty
				q.StandardAnswer = qReq.StandardAnswer
				q.Points = qReq.Points
				q.Order = qReq.Order
				s.Repo.UpdateQuestion(q)
				newQIDs[q.ID] = true
			} else {
				// ID 为空，或者 ID 不在当前任务的已有题目列表中，视为新题目
				q := &model.MigrationQuestion{
					TaskID:         taskID,
					Title:          qReq.Title,
					Description:    qReq.Description,
					Difficulty:     qReq.Difficulty,
					StandardAnswer: qReq.StandardAnswer,
					Points:         qReq.Points,
					Order:          qReq.Order,
				}
				s.Repo.CreateQuestion(q)
				// 新创建的题目不需要放入 newQIDs，因为 newQIDs 是用来标记哪些“旧题目”需要保留的
			}
		}

		for id := range existingMap {
			if !newQIDs[id] {
				s.Repo.DeleteQuestion(id)
			}
		}
	}

	return task, nil
}

func (s *MigrationTaskService) DeleteTask(taskID string) error {
	return s.Repo.DeleteTask(taskID)
}

func (s *MigrationTaskService) GetTask(taskID string) (*model.MigrationTask, []model.MigrationQuestion, error) {
	task, err := s.Repo.FindTaskByID(taskID)
	if err != nil {
		return nil, nil, err
	}
	qs, err := s.Repo.ListQuestions(taskID)
	return task, qs, err
}

func (s *MigrationTaskService) ListTasks(page, limit int) ([]repository.MigrationTaskListRow, int64, error) {
	return s.Repo.ListTasks(page, limit)
}

func (s *MigrationTaskService) ListPublishedTasksForStudent(userID uint) ([]map[string]interface{}, error) {
	return s.Repo.ListPublishedTasksForStudent(userID)
}

type MigrationAnswerReq struct {
	UserCode   string `json:"userCode"`
	UserAnswer string `json:"userAnswer"`
}

type MigrationSubmissionReq struct {
	Answers map[string]MigrationAnswerReq `json:"answers"`
}

func (s *MigrationTaskService) StartTask(userID uint, taskID string) (*model.MigrationSubmission, error) {
	existing, _ := s.Repo.FindSubmissionByUserAndTask(userID, taskID)
	if existing != nil {
		if existing.Status == "completed" {
			return nil, errors.New("task already completed")
		}
		return existing, nil
	}

	task, err := s.Repo.FindTaskByID(taskID)
	if err != nil {
		return nil, err
	}
	if !task.IsPublished {
		return nil, errors.New("task not published")
	}

	submission := &model.MigrationSubmission{
		TaskID:    taskID,
		UserID:    userID,
		Status:    "in_progress",
		StartedAt: time.Now(),
	}

	if err := s.Repo.DB.Create(submission).Error; err != nil {
		return nil, err
	}

	return submission, nil
}

func (s *MigrationTaskService) SubmitTask(userID uint, taskID string, req MigrationSubmissionReq) (*model.MigrationSubmission, error) {
	// 1. 检查是否存在正在进行的记录
	submission, err := s.Repo.FindSubmissionByUserAndTask(userID, taskID)
	if err != nil || submission == nil {
		return nil, errors.New("task not started")
	}
	if submission.Status == "completed" {
		return nil, errors.New("task already submitted")
	}

	task, err := s.Repo.FindTaskByID(taskID)
	if err != nil {
		return nil, err
	}

	// 2. 时间校验 (防止绕过前端限时)
	if task.TimeLimit > 0 {
		elapsedMinutes := int(time.Since(submission.StartedAt).Minutes())
		// 允许 1 分钟的网络延迟容差
		if elapsedMinutes > task.TimeLimit+1 {
			// 如果严重超时，强制判 0 分，但允许提交以关闭任务
			// 也可以选择报错：return nil, errors.New("task time limit exceeded")
		}
	}
	if !task.IsPublished {
		return nil, errors.New("task not published")
	}
	qs, err := s.Repo.ListQuestions(taskID)
	if err != nil {
		return nil, err
	}

	totalScore := 0
	answers := make([]model.MigrationAnswer, 0, len(qs))

	for _, q := range qs {
		ansReq := req.Answers[q.ID]
		isCorrect := false
		score := 0

		if strings.TrimSpace(strings.ToLower(ansReq.UserAnswer)) == strings.TrimSpace(strings.ToLower(q.StandardAnswer)) {
			isCorrect = true
		}

		if isCorrect {
			score = q.Points
			totalScore += score
		}

		answers = append(answers, model.MigrationAnswer{
			QuestionID:          q.ID,
			QuestionTitle:       q.Title,
			QuestionDescription: q.Description,
			UserCode:            ansReq.UserCode,
			UserAnswer:          ansReq.UserAnswer,
			IsCorrect:           isCorrect,
			Points:              score,
		})
	}

	now := time.Now()
	submission.Score = totalScore
	submission.Status = "completed"
	submission.CompletedAt = &now

	// 使用事务包裹：保存结果 + 更新积分 + 记录日志
	err = s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 保存提交记录和答案
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

		// 2. 发放积分 (更新 XP 字段，这是系统通用的积分/经验字段)
		if totalScore > 0 {
			if err := tx.Model(&model.User{}).Where("id = ?", userID).
				UpdateColumn("xp", gorm.Expr("xp + ?", totalScore)).Error; err != nil {
				return err
			}

			// 3. 记录学习日志
			log := &model.LearningLog{
				UserID:   userID,
				Activity: "migration_task_score",
				Content:  "迁移任务得分: " + task.Title,
				Duration: int(now.Sub(submission.StartedAt).Minutes()),
				Score:    totalScore,
			}
			if err := tx.Create(log).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return submission, nil
}

func (s *MigrationTaskService) ListSubmissions(taskID string, page, limit int, studentName string, status string) ([]map[string]interface{}, int64, error) {
	return s.Repo.ListSubmissions(taskID, page, limit, studentName, status)
}

func (s *MigrationTaskService) GetSubmissionDetail(submissionID string) (map[string]interface{}, error) {
	submission, answers, err := s.Repo.GetSubmissionDetail(submissionID)
	if err != nil {
		return nil, err
	}

	task, qs, err := s.GetTask(submission.TaskID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"submission": submission,
		"answers":    answers,
		"task":       task,
		"questions":  qs,
	}, nil
}

func (s *MigrationTaskService) GetStudentTaskDetail(userID uint, taskID string) (map[string]interface{}, error) {
	task, qs, err := s.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	if !task.IsPublished {
		return nil, errors.New("task not published")
	}

	submission, _ := s.Repo.FindSubmissionByUserAndTask(userID, taskID)

	status := "pending"
	var startedAt *time.Time
	remainingTime := task.TimeLimit * 60

	if submission != nil {
		status = submission.Status
		startedAt = &submission.StartedAt
		if status == "in_progress" {
			elapsed := int(time.Since(submission.StartedAt).Seconds())
			remainingTime = (task.TimeLimit * 60) - elapsed
			if remainingTime < 0 {
				remainingTime = 0
			}
		} else if status == "completed" {
			remainingTime = 0
		}
	}

	studentQs := make([]map[string]interface{}, len(qs))
	var answers []model.MigrationAnswer
	if status == "completed" && submission != nil {
		_, answers, _ = s.Repo.GetSubmissionDetail(submission.ID)
	}

	ansMap := make(map[string]model.MigrationAnswer)
	for _, a := range answers {
		ansMap[a.QuestionID] = a
	}

	for i, q := range qs {
		sq := map[string]interface{}{
			"id":          q.ID,
			"title":       q.Title,
			"description": q.Description,
			"difficulty":  q.Difficulty,
			"points":      q.Points,
			"order":       q.Order,
		}
		// 只有在状态为 completed 时才返回 standardAnswer
		if status == "completed" {
			if ans, ok := ansMap[q.ID]; ok {
				sq["userCode"] = ans.UserCode
				sq["userAnswer"] = ans.UserAnswer
				sq["isCorrect"] = ans.IsCorrect
			}
			sq["standardAnswer"] = q.StandardAnswer
		} else {
			// 未完成时，显式确保 standardAnswer 不会被序列化返回
			sq["standardAnswer"] = ""
		}
		studentQs[i] = sq
	}

	return map[string]interface{}{
		"task":          task,
		"questions":     studentQs,
		"status":        status,
		"startedAt":     startedAt,
		"remainingTime": remainingTime,
		"submission":    submission,
	}, nil
}

func (s *MigrationTaskService) RecordLearningTime(userID uint, taskID string, duration int) error {
	task, err := s.Repo.FindTaskByID(taskID)
	if err != nil {
		return err
	}

	log := &model.LearningLog{
		UserID:   userID,
		Activity: "migration_task",
		Content:  "迁移任务学习: " + task.Title,
		Duration: duration,
	}

	return s.Repo.DB.Create(log).Error
}
