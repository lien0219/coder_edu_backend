package model

type Achievement struct {
	BaseModel
	UserID   uint   `gorm:"index;type:bigint unsigned"`
	Name     string `gorm:"size:100;not null"`
	Icon     string `gorm:"size:255"`
	EarnedXP int    `gorm:"default:0"`
}

func (Achievement) TableName() string {
	return "achievements"
}
