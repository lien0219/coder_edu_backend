package model

import "time"

type LearningPath struct {
	Customized bool                 `json:"customized"`
	Modules    []LearningPathModule `json:"modules"`
}

type LearningPathModule struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Order int    `json:"order"`
}

type DiagnosticTest struct {
	Completed  bool     `json:"completed"`
	Strengths  []string `json:"strengths"`
	Weaknesses []string `json:"weaknesses"`
	Experience string   `json:"experience"`
}

type LearningGoal struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	TargetDate  time.Time `json:"targetDate"`
}
type LearningQuestion struct {
	ID      uint     `json:"id"`
	Text    string   `json:"text"`
	Options []string `json:"options"`
	Answer  int      `json:"answer"`
}
type Quiz struct {
	ID          uint               `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Score       int                `json:"score"`
	Total       int                `json:"total"`
	Completed   bool               `json:"completed"`
	Questions   []LearningQuestion `json:"questions"`
}

type PersonalizedRecommendation struct {
	TimeManagement       string   `json:"timeManagement"`
	FocusAreas           []string `json:"focusAreas"`
	CommunitySuggestions []string `json:"communitySuggestions"`
	ReviewTopics         []string `json:"reviewTopics"`
	ChallengeTasks       []string `json:"challengeTasks"`
}
