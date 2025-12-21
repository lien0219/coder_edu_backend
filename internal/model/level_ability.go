package model

// LevelAbility 关联表：关卡 <-> 能力
type LevelAbility struct {
	BaseModel
	LevelID   uint `gorm:"index;type:bigint unsigned" json:"levelId"`
	AbilityID uint `gorm:"index;type:bigint unsigned" json:"abilityId"`
}

func (LevelAbility) TableName() string {
	return "level_abilities"
}
