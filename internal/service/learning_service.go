package service

import (
	"coder_edu_backend/internal/model"
	"coder_edu_backend/internal/repository"
	"time"

	"gorm.io/gorm"
)

type LearningService struct {
	ModuleRepo      *repository.ModuleRepository
	TaskRepo        *repository.TaskRepository
	ResourceRepo    *repository.ResourceRepository
	ProgressRepo    *repository.ProgressRepository
	LearningLogRepo *repository.LearningLogRepository
	QuizRepo        *repository.QuizRepository
	DB              *gorm.DB
}

func NewLearningService(
	moduleRepo *repository.ModuleRepository,
	taskRepo *repository.TaskRepository,
	resourceRepo *repository.ResourceRepository,
	progressRepo *repository.ProgressRepository,
	learningLogRepo *repository.LearningLogRepository,
	quizRepo *repository.QuizRepository,
	db *gorm.DB,
) *LearningService {
	return &LearningService{
		ModuleRepo:      moduleRepo,
		TaskRepo:        taskRepo,
		ResourceRepo:    resourceRepo,
		ProgressRepo:    progressRepo,
		LearningLogRepo: learningLogRepo,
		QuizRepo:        quizRepo,
		DB:              db,
	}
}

type PreClassContent struct {
	DiagnosticTest model.DiagnosticTest `json:"diagnosticTest"`
	LearningGoals  []model.LearningGoal `json:"learningGoals"`
	LearningPath   model.LearningPath   `json:"learningPath"`
	Resources      []model.Resource     `json:"resources"`
}

type LearningPathModule struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Order int    `json:"order"`
}

type InClassContent struct {
	TaskChain        []TaskChainItem  `json:"taskChain"`
	RealTimeFeedback RealTimeFeedback `json:"realTimeFeedback"`
	Collaboration    Collaboration    `json:"collaboration"`
	CodeEditor       CodeEditor       `json:"codeEditor"`
}

type TaskChainItem struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"` // completed, in_progress, pending
	Order       int    `json:"order"`
}

type RealTimeFeedback struct {
	Errors    []FeedbackItem `json:"errors"`
	Warnings  []FeedbackItem `json:"warnings"`
	Successes []FeedbackItem `json:"successes"`
}

type FeedbackItem struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

type Collaboration struct {
	Messages []ChatMessage `json:"messages"`
}

type ChatMessage struct {
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type CodeEditor struct {
	DefaultCode string `json:"defaultCode"`
	Output      string `json:"output"`
}

type PostClassContent struct {
	LearningJournal LearningJournal `json:"learningJournal"`
	Quizzes         []model.Quiz    `json:"quizzes"`
	TransferTasks   []TransferTask  `json:"transferTasks"`
	ReflectionGuide ReflectionGuide `json:"reflectionGuide"`
}

type LearningJournal struct {
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

type TransferTask struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Difficulty  string `json:"difficulty"` // A, B, C, etc.
}

type ReflectionGuide struct {
	Questions []string `json:"questions"`
}

type LearningLogRequest struct {
	Content    string   `json:"content" binding:"required"`
	Tags       []string `json:"tags"`
	Insights   []string `json:"insights"`
	Challenges []string `json:"challenges"`
	NextSteps  []string `json:"nextSteps"`
}

type QuizResult struct {
	Score   int `json:"score"`
	Total   int `json:"total"`
	Correct int `json:"correct"`
	Wrong   int `json:"wrong"`
}

type QuizSubmission struct {
	Answers map[uint]int `json:"answers"` // questionID -> answerIndex
}

func (s *LearningService) GetPreClassContent(userID uint) (*PreClassContent, error) {
	// 获取诊断测试结果
	diagnostic, err := s.ProgressRepo.GetDiagnosticTest(userID)
	if err != nil {
		return nil, err
	}

	// 获取学习目标
	goals, err := s.ProgressRepo.GetLearningGoals(userID)
	if err != nil {
		return nil, err
	}

	// 获取学习路径
	path, err := s.ModuleRepo.GetLearningPath(userID)
	if err != nil {
		return nil, err
	}

	// 获取课前资源
	resources, err := s.ResourceRepo.FindByModuleType("pre_class")
	if err != nil {
		return nil, err
	}

	return &PreClassContent{
		DiagnosticTest: diagnostic,
		LearningGoals:  goals,
		LearningPath:   path,
		Resources:      resources,
	}, nil
}

func (s *LearningService) GetInClassContent(userID uint) (*InClassContent, error) {
	// 获取任务链
	tasks, err := s.TaskRepo.FindByModuleTypeAndUser("in_class", userID)
	if err != nil {
		return nil, err
	}

	taskChain := make([]TaskChainItem, len(tasks))
	for i, task := range tasks {
		taskChain[i] = TaskChainItem{
			ID:          task.ID,
			Title:       task.Title,
			Description: task.Description,
			Status:      string(task.Status),
			Order:       task.Order,
		}
	}

	// 获取实时反馈（简化实现）
	feedback := RealTimeFeedback{
		Errors: []FeedbackItem{
			{
				Type:    "error",
				Message: "Missing semicolon on line 5",
				Line:    5,
			},
		},
		Warnings: []FeedbackItem{
			{
				Type:    "warning",
				Message: "Unused variable 'temp' declared on line 12",
				Line:    12,
			},
		},
		Successes: []FeedbackItem{
			{
				Type:    "success",
				Message: "Test Case 1: Passed. Output matches expected.",
			},
		},
	}

	// 获取协作消息（简化实现）
	collaboration := Collaboration{
		Messages: []ChatMessage{
			{
				Author:    "Prof. Ada",
				Content:   "Great progress everyone! Let's discuss the approach for Task 3's algorithm.",
				Timestamp: time.Now().Add(-10 * time.Minute),
			},
			{
				Author:    "You",
				Content:   "I'm thinking of using a recursive solution for Task 3, but I'm unsure about this base case.",
				Timestamp: time.Now().Add(-5 * time.Minute),
			},
			{
				Author:    "Alex M.",
				Content:   "For Task 3, consider an iterative approach with a loop. It might be more straightforward for this specific problem.",
				Timestamp: time.Now().Add(-2 * time.Minute),
			},
		},
	}

	// 代码编辑器默认代码
	codeEditor := CodeEditor{
		DefaultCode: `#include <stdio.h>
int main() {
    printf("Hello, SDL Learning\\n");
    return 0;
}`,
		Output: "Hello, SDL Learning Website!",
	}

	return &InClassContent{
		TaskChain:        taskChain,
		RealTimeFeedback: feedback,
		Collaboration:    collaboration,
		CodeEditor:       codeEditor,
	}, nil
}

func (s *LearningService) GetPostClassContent(userID uint) (*PostClassContent, error) {
	// 获取学习日志
	var journal model.LearningLog
	err := s.DB.Where("user_id = ?", userID).Order("created_at DESC").First(&journal).Error
	if err != nil {
		return nil, err
	}

	// 获取测验
	quizzes, err := s.QuizRepo.FindByModuleType("post_class")
	if err != nil {
		return nil, err
	}

	// 获取迁移任务
	tasks, err := s.TaskRepo.FindTransferTasks(userID)
	if err != nil {
		return nil, err
	}

	transferTasks := make([]TransferTask, len(tasks))
	for i, task := range tasks {
		transferTasks[i] = TransferTask{
			ID:          task.ID,
			Title:       task.Title,
			Description: task.Description,
			Status:      string(task.Status),
			Difficulty:  task.Difficulty,
		}
	}

	// 反思指南问题
	reflectionGuide := ReflectionGuide{
		Questions: []string{
			"你今天学习了哪些新的概念或技能？",
			"在解决问题时，哪些部分让你感到特别有挑战吗？你是如何克服的？",
			"今天的学习如何与你之前的知识联系起来？它改变了你对某个概念的理解吗？",
			"有没有什么你仍然感到困惑的地方？下一步你需要学习什么？",
			"如果你能重写，你会如何改进今天的学习过程和方法？",
			"总结一下今天的学习成果，并思考它的实际应用。",
		},
	}

	return &PostClassContent{
		LearningJournal: LearningJournal{
			Content: journal.Content,
			Tags:    journal.Tags,
		},
		Quizzes:         quizzes,
		TransferTasks:   transferTasks,
		ReflectionGuide: reflectionGuide,
	}, nil
}

func (s *LearningService) SubmitLearningLog(userID uint, req LearningLogRequest) error {
	log := &model.LearningLog{
		UserID:     userID,
		Content:    req.Content,
		Tags:       req.Tags,
		Insights:   req.Insights,
		Challenges: req.Challenges,
		NextSteps:  req.NextSteps,
	}

	return s.LearningLogRepo.Create(log)
}

func (s *LearningService) SubmitQuiz(userID uint, quizID uint, submission QuizSubmission) (*QuizResult, error) {
	// 获取测验
	quiz, err := s.QuizRepo.FindByID(quizID)
	if err != nil {
		return nil, err
	}

	// 计算得分
	score := 0
	for qid, answer := range submission.Answers {
		for _, question := range quiz.Questions {
			if question.ID == qid && question.Answer == answer {
				score++
				break
			}
		}
	}

	// 保存测验结果
	result := &model.QuizResult{
		UserID:    userID,
		QuizID:    quizID,
		Score:     score,
		Total:     len(quiz.Questions),
		Answers:   submission.Answers,
		Completed: true,
	}

	err = s.QuizRepo.SaveResult(result)
	if err != nil {
		return nil, err
	}

	return &QuizResult{
		Score:   score,
		Total:   len(quiz.Questions),
		Correct: score,
		Wrong:   len(quiz.Questions) - score,
	}, nil
}
