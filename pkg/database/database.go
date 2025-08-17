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
	)

	if err != nil {
		return nil, err
	}

	log.Println("Database migration completed")
	return db, nil
}
