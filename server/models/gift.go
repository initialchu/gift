package models

import (
	"time"

	"gorm.io/gorm"
)

// 礼薄模型(数据库表结构)

type GiftBook struct {
	gorm.Model

	EventName string       `gorm:"not null" json:"event_name"`
	EventDate time.Time    `json:"event_date"`
	Direction string       `gorm:"type:varchar(4);default:来;not null" json:"direction"`
	CreatedBy string       `gorm:"not null" json:"created_by"`
	Records   []GiftRecord `gorm:"foreignKey:GiftBookID" json:"records,omitempty"`
}

// 礼金模型(数据库表结构)
type GiftRecord struct {
	gorm.Model
	GiftBookID uint    `gorm:"not null;index" json:"gift_book_id"`
	CardID     uint    `gorm:"index" json:"card_id"`
	PersonName string  `gorm:"type:varchar(64);not null" json:"person_name"`
	Amount     float64 `gorm:"type:decimal(10,2);not null" json:"amount"`
	Address    string  `gorm:"type:varchar(255)" json:"address,omitempty"`
	GiftNote   string  `gorm:"type:varchar(255)" json:"gift_note,omitempty"`
}
