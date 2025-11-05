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
		Logger: logger.Default.LogMode(logger.Info),
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

	return db, nil
}
