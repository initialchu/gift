package models

import "gorm.io/gorm"

// Card 人情卡片 —— 按「人」归集，同名可建多张卡片区分不同的人
type Card struct {
	gorm.Model
	PersonName string       `gorm:"type:varchar(64);not null;index" json:"person_name"`
	Note       string       `gorm:"type:varchar(255)" json:"note,omitempty"`
	Records    []GiftRecord `gorm:"foreignKey:CardID" json:"records,omitempty"`
}
