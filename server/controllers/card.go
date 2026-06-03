package controllers

import (
	"giftmemo/global"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CardSummary 人情卡片汇总（每人一张卡）
type CardSummary struct {
	CardID          uint    `json:"card_id"`
	PersonName      string  `json:"person_name"`
	ReceivedCount   int64   `json:"received_count"`
	ReceivedAmount  float64 `json:"received_amount"`
	GivenCount      int64   `json:"given_count"`
	GivenAmount     float64 `json:"given_amount"`
	NetAmount       float64 `json:"net_amount"`
}

// CardDetail 某人的往来明细（含礼薄信息，供前端跳转）
type CardDetail struct {
	ID         uint    `json:"id"`
	GiftBookID uint    `json:"gift_book_id"`
	PersonName string  `json:"person_name"`
	Amount     float64 `json:"amount"`
	Address    string  `json:"address,omitempty"`
	GiftNote   string  `json:"gift_note,omitempty"`
	EventName  string  `json:"event_name"`
	EventDate  string  `json:"event_date"`
	Direction  string  `json:"direction"`
}

// GetCards 获取所有卡片汇总（按 card_id 分组，避免同名混淆）
func GetCards(c *gin.Context) {
	var cards []CardSummary
	err := global.DB.Raw(`
		SELECT
			c.id   AS card_id,
			c.person_name,
			COUNT(CASE WHEN gb.direction = '来' THEN 1 END)       AS received_count,
			COALESCE(SUM(CASE WHEN gb.direction = '来' THEN gr.amount ELSE 0 END), 0) AS received_amount,
			COUNT(CASE WHEN gb.direction = '去' THEN 1 END)       AS given_count,
			COALESCE(SUM(CASE WHEN gb.direction = '去' THEN gr.amount ELSE 0 END), 0) AS given_amount,
			COALESCE(SUM(CASE WHEN gb.direction = '来' THEN gr.amount ELSE -gr.amount END), 0) AS net_amount
		FROM gift_records AS gr
		JOIN gift_books AS gb ON gr.gift_book_id = gb.id
		JOIN cards AS c ON gr.card_id = c.id
		GROUP BY c.id, c.person_name
		ORDER BY c.person_name
	`).Scan(&cards).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取卡片失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cards": cards})
}

// GetCardDetail 获取某张卡片的全部往来明细
func GetCardDetail(c *gin.Context) {
	cardID := c.Query("card_id")
	if cardID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 card_id 参数"})
		return
	}

	var records []CardDetail
	err := global.DB.Table("gift_records").
		Select("gift_records.*, gift_books.event_name, gift_books.event_date, gift_books.direction").
		Joins("JOIN gift_books ON gift_records.gift_book_id = gift_books.id").
		Where("gift_records.card_id = ?", cardID).
		Order("gift_books.event_date DESC").
		Scan(&records).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取明细失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"records": records})
}
