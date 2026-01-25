package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type PostClassTestService struct {
	Repo    *repository.PostClassTestRepository
	UserSvc *UserService
}

func NewPostClassTestService(repo *repository.PostClassTestRepository, userSvc *UserService) *PostClassTestService {
	return &PostClassTestService{Repo: repo, UserSvc: userSvc}
}

type PostClassTestQuestionReq struct {
	ID           string          `json:"id"`
	QuestionType string          `json:"questionType" binding:"required"`
	Content      string          `json:"content" binding:"required"`
	Options      json.RawMessage `json:"options"`
	Answer       string          `json:"answer" binding:"required"`
	Points       int             `json:"points"`
	RewardXP     int             `json:"rewardXp"`
	Explanation  string          `json:"explanation"`
	Order        int             `json:"order"`
}

type PostClassTestReq struct {
	Title       *string                     `json:"title"`
	Description *string                     `json:"description"`
	TimeLimit   *int                        `json:"timeLimit"`
	IsPublished *bool                       `json:"isPublished"`
	Questions   *[]PostClassTestQuestionReq `json:"questions"`
}

func (s *PostClassTestService) CreateTest(creatorID uint, req PostClassTestReq) (*model.PostClassTest, error) {
	if req.Title == nil || *req.Title == "" {
		return nil, errors.New("title is required")
	}

	test := &model.PostClassTest{
		Title:     *req.Title,
		CreatorID: creatorID,
	}

	if req.Description != nil {
		test.Description = *req.Description
	}
	if req.TimeLimit != nil {
		test.TimeLimit = *req.TimeLimit
	}
	if req.IsPublished != nil {
		test.IsPublished = *req.IsPublished
	}

	if err := s.Repo.CreateTest(test); err != nil {
		return nil, err
	}

	if test.IsPublished {
		_ = s.Repo.UnpublishAllExcept(test.ID)
	}

	if req.Questions != nil {
		for _, qReq := range *req.Questions {
			q := &model.PostClassTestQuestion{
				TestID:       test.ID,
				QuestionType: qReq.QuestionType,
				Content:      qReq.Content,
				Options:      qReq.Options,
				Answer:       qReq.Answer,
				Points:       qReq.Points,
				RewardXP:     qReq.RewardXP,
				Explanation:  qReq.Explanation,
				Order:        qReq.Order,
			}
			if err := s.Repo.CreateQuestion(q); err != nil {
				return nil, err
			}
		}
	}

	return test, nil
}

func (s *PostClassTestService) UpdateTest(testID string, req PostClassTestReq) (*model.PostClassTest, error) {
	test, err := s.Repo.FindTestByID(testID)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		test.Title = *req.Title
	}
	if req.Description != nil {
		test.Description = *req.Description
	}
	if req.TimeLimit != nil {
		test.TimeLimit = *req.TimeLimit
	}
	if req.IsPublished != nil {
		test.IsPublished = *req.IsPublished
	}

	if err := s.Repo.UpdateTest(test); err != nil {
		return nil, err
	}

	if test.IsPublished {
		_ = s.Repo.UnpublishAllExcept(test.ID)
	}

	if req.Questions != nil {
		existingQs, _ := s.Repo.ListQuestions(testID)
		existingMap := make(map[string]*model.PostClassTestQuestion)
		for i := range existingQs {
			existingMap[existingQs[i].ID] = &existingQs[i]
		}

		newQIDs := make(map[string]bool)
		for _, qReq := range *req.Questions {
			if qReq.ID != "" {
				if q, ok := existingMap[qReq.ID]; ok {
					q.QuestionType = qReq.QuestionType
					q.Content = qReq.Content
					q.Options = qReq.Options
					q.Answer = qReq.Answer
					q.Points = qReq.Points
					q.RewardXP = qReq.RewardXP
					q.Explanation = qReq.Explanation
					q.Order = qReq.Order
					s.Repo.UpdateQuestion(q)
					newQIDs[q.ID] = true
				}
			} else {
				q := &model.PostClassTestQuestion{
					TestID:       testID,
					QuestionType: qReq.QuestionType,
					Content:      qReq.Content,
					Options:      qReq.Options,
					Answer:       qReq.Answer,
					Points:       qReq.Points,
					RewardXP:     qReq.RewardXP,
					Explanation:  qReq.Explanation,
					Order:        qReq.Order,
				}
				s.Repo.CreateQuestion(q)
			}
		}

		for id := range existingMap {
			if !newQIDs[id] {
				s.Repo.DeleteQuestion(id)
			}
		}
	}

	return test, nil
}

func (s *PostClassTestService) DeleteTest(testID string) error {
	return s.Repo.DeleteTest(testID)
}

func (s *PostClassTestService) GetTest(testID string) (*model.PostClassTest, []model.PostClassTestQuestion, error) {
	test, err := s.Repo.FindTestByID(testID)
	if err != nil {
		return nil, nil, err
	}
	qs, err := s.Repo.ListQuestions(testID)
	return test, qs, err
}

func (s *PostClassTestService) ListTests(page, limit int) ([]repository.PostClassTestListRow, int64, error) {
	return s.Repo.ListTests(page, limit)
}

func (s *PostClassTestService) GetPublishedTestForStudent(userID uint) (*repository.PublishedTestForStudent, error) {
	return s.Repo.GetPublishedTestForStudent(userID)
}

type PostClassTestAnswerReq struct {
	Result string `json:"result"` // 用于比对得分的结果
	Code   string `json:"code"`   // 如果是代码题，提交的源代码
}

type PostClassTestSubmissionReq struct {
	Answers   map[string]PostClassTestAnswerReq `json:"answers"`
	IsTimeout bool                              `json:"isTimeout"`
}

func (s *PostClassTestService) SubmitTest(userID uint, testID string, req PostClassTestSubmissionReq) (*model.PostClassTestSubmission, error) {
	// 1. 检查是否存在正在进行的记录
	submission, err := s.Repo.FindSubmissionByUserAndTest(userID, testID)
	if err != nil || submission == nil {
		return nil, errors.New("test not started")
	}
	if submission.Status == "completed" {
		return nil, errors.New("test already submitted")
	}

	// 2. 获取试卷和题目
	test, err := s.Repo.FindTestByID(testID)
	if err != nil {
		return nil, err
	}
	qs, err := s.Repo.ListQuestions(testID)
	if err != nil {
		return nil, err
	}

	// 3. 评分和计算积分
	totalScore := 0
	totalXP := 0
	answers := make([]model.PostClassTestAnswer, 0, len(qs))

	for _, q := range qs {
		ansReq := req.Answers[q.ID]
		userAns := ansReq.Result
		isCorrect := false
		score := 0

		// 评分逻辑：比对结果
		switch q.QuestionType {
		case "single_choice", "multiple_choice", "true_false", "fill_blank", "code", "programming", "essay":
			// 比对去除首尾空格后的结果（忽略大小写）
			if strings.TrimSpace(strings.ToLower(userAns)) == strings.TrimSpace(strings.ToLower(q.Answer)) {
				isCorrect = true
			}
		}

		if isCorrect {
			score = q.Points
			totalScore += score
			totalXP += q.RewardXP
		}

		// 存储内容：如果是代码题，UserAnswer 存储 JSON，包含代码和结果
		storageAnswer := userAns
		if q.QuestionType == "code" || q.QuestionType == "programming" {
			ansJson, _ := json.Marshal(map[string]string{
				"result": userAns,
				"code":   ansReq.Code,
			})
			storageAnswer = string(ansJson)
		}

		answers = append(answers, model.PostClassTestAnswer{
			QuestionID:      q.ID,
			QuestionType:    q.QuestionType,
			QuestionContent: q.Content,
			QuestionOptions: q.Options,
			UserAnswer:      storageAnswer,
			IsCorrect:       isCorrect,
			Score:           score,
		})
	}

	// 4. 更新记录
	now := time.Now()
	submission.Score = totalScore
	submission.RewardXP = totalXP
	submission.Status = "completed"
	submission.IsTimeout = req.IsTimeout
	submission.CompletedAt = &now

	if err := s.Repo.UpdateSubmissionWithAnswers(submission, answers); err != nil {
		return nil, err
	}

	// 5. 更新用户积分 (实现同一学生同一测试题不能重复获得积分)
	if totalXP > 0 {
		// 检查学习日志中是否已有该试卷的得分记录
		var count int64
		s.Repo.DB.Model(&model.LearningLog{}).
			Where("user_id = ? AND activity = ? AND content LIKE ?", userID, "post_class_test_score", "%试卷ID: "+testID+"%").
			Count(&count)

		if count == 0 {
			_ = s.UserSvc.UpdateUserPoints(userID, totalXP)
			// 记录得分日志
			_ = s.Repo.DB.Create(&model.LearningLog{
				UserID:   userID,
				Activity: "post_class_test_score",
				Content:  "课后测试得分: " + test.Title + " (试卷ID: " + testID + ")",
				Duration: 0,
				Score:    totalScore,
			})
		}
	}

	return submission, nil
}

type StudentPostClassTestQuestion struct {
	ID           string          `json:"id"`
	QuestionType string          `json:"questionType"`
	Content      string          `json:"content"`
	Options      json.RawMessage `json:"options"`
	Points       int             `json:"points"`
	Order        int             `json:"order"`
	// status == 'completed' 时返回
	IsCorrect   *bool   `json:"isCorrect,omitempty"`
	UserAnswer  *string `json:"userAnswer,omitempty"`
	StudentCode *string `json:"studentCode,omitempty"` // 学生编写的代码
	Answer      *string `json:"answer,omitempty"`      // 标准答案
	Explanation *string `json:"explanation,omitempty"` // 题目解析
}

type StudentPostClassTestDetail struct {
	ID            string                         `json:"id"`
	Title         string                         `json:"title"`
	Description   string                         `json:"description"`
	TimeLimit     int                            `json:"timeLimit"` // 总限时（分钟）
	QuestionCount int                            `json:"questionCount"`
	Status        string                         `json:"status"`        // pending, in_progress, completed
	StartedAt     *time.Time                     `json:"startedAt"`     // 开始答题时间
	RemainingTime int                            `json:"remainingTime"` // 剩余秒数
	Score         *int                           `json:"score,omitempty"`
	RewardXP      *int                           `json:"rewardXp,omitempty"`
	Questions     []StudentPostClassTestQuestion `json:"questions"`
}

func (s *PostClassTestService) StartTest(userID uint, testID string) (*model.PostClassTestSubmission, error) {
	// 1. 检查是否已经存在（已开始或已完成）
	existing, _ := s.Repo.FindSubmissionByUserAndTest(userID, testID)
	if existing != nil {
		return existing, nil
	}

	// 2. 获取试卷信息
	test, err := s.Repo.FindTestByID(testID)
	if err != nil {
		return nil, err
	}
	if !test.IsPublished {
		return nil, errors.New("test not published")
	}

	// 3. 创建 in_progress 记录
	submission := &model.PostClassTestSubmission{
		TestID:    testID,
		UserID:    userID,
		Status:    "in_progress",
		StartedAt: time.Now(),
	}

	if err := s.Repo.CreateSubmission(submission); err != nil {
		return nil, err
	}

	return submission, nil
}

func (s *PostClassTestService) GetStudentTestDetail(userID uint, testID string) (*StudentPostClassTestDetail, error) {
	test, err := s.Repo.FindTestByID(testID)
	if err != nil {
		return nil, err
	}

	if !test.IsPublished {
		return nil, errors.New("test not published or not accessible")
	}

	qs, err := s.Repo.ListQuestions(testID)
	if err != nil {
		return nil, err
	}

	submission, _ := s.Repo.FindSubmissionByUserAndTest(userID, testID)
	status := "pending"
	var startedAt *time.Time
	remainingTime := test.TimeLimit * 60 // 默认总秒数

	var answers []model.PostClassTestAnswer
	if submission != nil {
		status = submission.Status
		startedAt = &submission.StartedAt

		if status == "in_progress" {
			// 计算剩余时间：总限时(秒) - 已过去时间
			elapsed := int(time.Since(submission.StartedAt).Seconds())
			remainingTime = (test.TimeLimit * 60) - elapsed
			if remainingTime < 0 {
				remainingTime = 0
			}
		} else if status == "completed" {
			remainingTime = 0
			// 如果是完成状态，加载答题记录
			_, answers, _ = s.Repo.GetSubmissionDetail(submission.ID)
		}
	}

	// 建立题目ID到答案的映射
	ansMap := make(map[string]model.PostClassTestAnswer)
	for _, a := range answers {
		ansMap[a.QuestionID] = a
	}

	studentQs := make([]StudentPostClassTestQuestion, len(qs))
	for i, q := range qs {
		sq := StudentPostClassTestQuestion{
			ID:           q.ID,
			QuestionType: q.QuestionType,
			Content:      q.Content,
			Options:      q.Options,
			Points:       q.Points,
			Order:        q.Order,
		}

		// 如果是完成状态，填充结果、标准答案和解析
		if status == "completed" {
			if ans, ok := ansMap[q.ID]; ok {
				isCorrect := ans.IsCorrect
				sq.IsCorrect = &isCorrect

				userAns := ans.UserAnswer
				// 如果是编程题，尝试解析出结果和代码
				if q.QuestionType == "code" || q.QuestionType == "programming" {
					var codeMap map[string]string
					if err := json.Unmarshal([]byte(ans.UserAnswer), &codeMap); err == nil {
						res := codeMap["result"]
						sq.UserAnswer = &res
						code := codeMap["code"]
						sq.StudentCode = &code
					} else {
						sq.UserAnswer = &userAns
					}
				} else {
					sq.UserAnswer = &userAns
				}
			}
			standardAns := q.Answer
			explanation := q.Explanation
			sq.Answer = &standardAns
			sq.Explanation = &explanation
		}

		studentQs[i] = sq
	}

	detail := &StudentPostClassTestDetail{
		ID:            test.ID,
		Title:         test.Title,
		Description:   test.Description,
		TimeLimit:     test.TimeLimit,
		QuestionCount: len(qs),
		Status:        status,
		StartedAt:     startedAt,
		RemainingTime: remainingTime,
		Questions:     studentQs,
	}

	// 如果完成，填充总分和积分
	if status == "completed" && submission != nil {
		score := submission.Score
		xp := submission.RewardXP
		detail.Score = &score
		detail.RewardXP = &xp
	}

	return detail, nil
}

func (s *PostClassTestService) RecordLearningTime(userID uint, testID string, duration int) error {
	test, err := s.Repo.FindTestByID(testID)
	if err != nil {
		return err
	}

	log := &model.LearningLog{
		UserID:   userID,
		Activity: "post_class_test",
		Content:  "课后测试学习: " + test.Title,
		Duration: duration,
	}

	return s.Repo.DB.Create(log).Error
}

func (s *PostClassTestService) ListSubmissions(testID string, page, limit int, studentName string, status string) ([]map[string]interface{}, int64, error) {
	return s.Repo.ListSubmissions(testID, page, limit, studentName, status)
}

func (s *PostClassTestService) GetSubmissionDetail(submissionID string) (map[string]interface{}, error) {
	submission, answers, err := s.Repo.GetSubmissionDetail(submissionID)
	if err != nil {
		return nil, err
	}

	test, qs, err := s.GetTest(submission.TestID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"submission": submission,
		"answers":    answers,
		"test":       test,
		"questions":  qs,
	}, nil
}

func (s *PostClassTestService) ResetStudentTest(submissionID string) error {
	return s.Repo.DeleteSubmission(submissionID)
}

func (s *PostClassTestService) BatchResetStudentTests(submissionIDs []string) error {
	return s.Repo.BatchDeleteSubmissions(submissionIDs)
}
