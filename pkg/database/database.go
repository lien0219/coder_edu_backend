package database

import (
	"coder_edu_backend/internal/config"
	"coder_edu_backend/internal/model"
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// var DB *gorm.DB

func InitDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
		cfg.Charset,
		cfg.ParseTime,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		return nil, err
	}

	log.Println("Database connection established")

	err = db.AutoMigrate(
		&model.User{},
		&model.Achievement{},
		&model.Resource{},
		&model.Task{},
		&model.Motivation{},
		&model.LearningModule{},
		&model.UserProgress{},
		&model.LearningLog{},
		&model.QuizResult{},
		&model.Goal{},
		&model.LearningSession{},
		&model.SkillAssessment{},
		&model.Post{},
		&model.Comment{},
		&model.Question{},
		&model.Answer{},
		&model.CProgrammingResource{},
		&model.ExerciseCategory{},
		&model.ExerciseQuestion{},
		&model.ExerciseSubmission{},
		&model.Checkin{},
		&model.ResourceCompletion{},
		&model.TeacherWeeklyTask{},
		&model.TaskItem{},
		&model.DailyTaskCompletion{},
		&model.Level{},
		&model.LevelVersion{},
		&model.LevelQuestion{},
		&model.LevelAttempt{},
		&model.Ability{},
		&model.LevelAbility{},
		&model.KnowledgeTag{},
		&model.LevelKnowledge{},
		&model.LevelAttemptQuestionTime{},
		&model.LevelAttemptAnswer{},
		&model.LevelAttemptQuestionScore{},
	)

	if err != nil {
		return nil, err
	}

	log.Println("Database migration completed")

	// 默认的激励短句
	var count int64
	db.Model(&model.Motivation{}).Count(&count)
	if count == 0 {
		defaultMotivations := []string{
			"您编写的每一行代码都是迈向精通的一步。继续编程！",
			"学习是唯一的财富，因为它可以被分享而不会减少。",
			"Consistency is the key to programming success.",
			"编程不是关于知道所有答案，而是关于知道如何找到它们。",
		}
		for i, content := range defaultMotivations {
			motivation := &model.Motivation{
				Content:         content,
				IsEnabled:       true,
				IsCurrentlyUsed: i == 0,
			}
			db.Create(motivation)
		}
	}

	// 默认知识点标签（如果为空则插入一些常用知识点）
	var ktCount int64
	db.Model(&model.KnowledgeTag{}).Count(&ktCount)
	if ktCount == 0 {
		defaultTags := []model.KnowledgeTag{
			{Code: "array", Name: "数组", Description: "数组与索引", Enabled: true},
			{Code: "loop", Name: "循环", Description: "for/while 循环", Enabled: true},
			{Code: "pointer", Name: "指针", Description: "指针与地址访问", Enabled: true},
			{Code: "recursion", Name: "递归", Description: "递归与分治", Enabled: true},
			{Code: "sort", Name: "排序", Description: "常见排序算法", Enabled: true},
			{Code: "search", Name: "查找", Description: "线性/二分查找", Enabled: true},
		}
		for _, t := range defaultTags {
			db.Create(&t)
		}
	}

	return db, nil
}
