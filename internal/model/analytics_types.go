package model

// MonthlyData 月度学习数据
type MonthlyData struct {
	Month            string  `json:"month"`
	ModulesCompleted int     `json:"modulesCompleted"`
	AverageScore     float64 `json:"averageScore"`
}

// WeekProgress 周学习进度
type WeekProgress struct {
	Week             string  `json:"week"`
	StudyTime        int     `json:"studyTime"`
	ModulesCompleted int     `json:"modulesCompleted"`
	AverageScore     float64 `json:"averageScore"`
}

// OverallProgress 总体进度
type OverallProgress struct {
	TotalModules     int     `json:"totalModules"`
	CompletedModules int     `json:"completedModules"`
	AverageScore     float64 `json:"averageScore"`
}

// LearningOverview 学习概览
type LearningOverview struct {
	TotalModules     int                `json:"totalModules"`
	CompletedModules int                `json:"completedModules"`
	AverageScore     float64            `json:"averageScore"`
	MonthlyProgress  []MonthlyData      `json:"monthlyProgress"`
	ModuleCompletion map[string]float64 `json:"moduleCompletion"` // 模块名称 -> 完成百分比
}

// LearningProgress 学习进度
type LearningProgress struct {
	Weeks []WeekProgress `json:"weeks"`
	Trend string         `json:"trend"` // improving, declining, stable
}

// ChallengeWeeklyData 挑战周数据
type ChallengeWeeklyData struct {
	Week           string  `json:"week"`
	AverageScore   float64 `json:"averageScore"`
	CompletedCount int     `json:"completedCount"`
}

// SkillRadar 技能雷达图
type SkillRadar struct {
	Skills            []string `json:"skills"`
	KnowledgeCoverage []int    `json:"knowledgeCoverage"` // 0-100
	ProblemSolving    []int    `json:"problemSolving"`    // 0-100
}

// AbilityRadarData 能力雷达图单项数据
type AbilityRadarData struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// AttemptCurveData 关卡尝试曲线单项数据
type AttemptCurveData struct {
	AttemptIndex int    `json:"attemptIndex"` // 第几次尝试
	Score        int    `json:"score"`        // 分数
	Date         string `json:"date"`         // 尝试日期
}

// LevelCurveResponse 关卡尝试趋势响应
type LevelCurveResponse struct {
	LevelID    uint               `json:"levelId"`
	LevelTitle string             `json:"levelTitle"`
	Attempts   []AttemptCurveData `json:"attempts"`
}
