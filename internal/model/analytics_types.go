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

// SkillRadar 技能雷达图
type SkillRadar struct {
	Skills            []string `json:"skills"`
	KnowledgeCoverage []int    `json:"knowledgeCoverage"` // 0-100
	ProblemSolving    []int    `json:"problemSolving"`    // 0-100
}
