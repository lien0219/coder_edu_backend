package repository

import (
	"coder_edu_backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type ProgressRepository struct {
	DB *gorm.DB
}

func NewProgressRepository(db *gorm.DB) *ProgressRepository {
	return &ProgressRepository{DB: db}
}

func (r *ProgressRepository) GetOverallProgress(userID uint) (*model.OverallProgress, error) {
	var totalModules int64
	err := r.DB.Model(&model.LearningModule{}).Count(&totalModules).Error
	if err != nil {
		return nil, err
	}

	var completedModules int64
	err = r.DB.Model(&model.UserProgress{}).
		Where("user_id = ? AND completed = ?", userID, true).
		Count(&completedModules).Error
	if err != nil {
		return nil, err
	}

	var averageScore float64
	err = r.DB.Model(&model.UserProgress{}).
		Where("user_id = ?", userID).
		Select("AVG(score)").
		Scan(&averageScore).Error
	if err != nil {
		return nil, err
	}

	return &model.OverallProgress{
		TotalModules:     int(totalModules),
		CompletedModules: int(completedModules),
		AverageScore:     averageScore,
	}, nil
}

func (r *ProgressRepository) GetMonthlyProgress(userID uint, months int) ([]model.MonthlyData, error) {
	// 实现获取月度进度数据的逻辑
	// 模拟数据
	now := time.Now()
	monthlyData := make([]model.MonthlyData, months)

	for i := 0; i < months; i++ {
		month := now.AddDate(0, -i, 0).Format("2006-01")
		monthlyData[i] = model.MonthlyData{
			Month:            month,
			ModulesCompleted: 2 + i%3,
			AverageScore:     75.0 + float64(i)*2.5,
		}
	}

	return monthlyData, nil
}

func (r *ProgressRepository) GetModuleCompletion(userID uint) (map[string]float64, error) {
	// 实现获取模块完成情况的逻辑
	// 模拟数据
	moduleCompletion := map[string]float64{
		"C语言基础":   85.0,
		"数据结构":    60.0,
		"指针和内存管理": 45.0,
		"文件操作":    70.0,
		"多线程编程":   30.0,
	}

	return moduleCompletion, nil
}

func (r *ProgressRepository) GetWeeklyProgress(userID uint, weeks int) ([]model.WeekProgress, error) {
	// 实现获取周进度数据的逻辑
	// 模拟数据
	now := time.Now()
	weeklyProgress := make([]model.WeekProgress, weeks)

	for i := 0; i < weeks; i++ {
		weekStart := now.AddDate(0, 0, -i*7).Format("2006-01-02")
		weekEnd := now.AddDate(0, 0, -i*7+6).Format("2006-01-02")
		weekLabel := weekStart + " 至 " + weekEnd

		weeklyProgress[i] = model.WeekProgress{
			Week:             weekLabel,
			StudyTime:        120 + i*30,
			ModulesCompleted: 1 + i%2,
			AverageScore:     70.0 + float64(i)*3.5,
		}
	}

	return weeklyProgress, nil
}
