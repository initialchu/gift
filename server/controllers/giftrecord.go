package controllers

import (
	"giftmemo/global"
	"giftmemo/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// 礼金记录控制器

// 添加礼金记录
func AddGift(c *gin.Context) {
	var giftRecord models.GiftRecord
	if err := c.ShouldBindJSON(&giftRecord); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 处理卡片关联：如果传了 card_id 就用，否则按 person_name 查找或创建卡片
	if giftRecord.CardID == 0 {
		var card models.Card
		global.DB.Where("person_name = ?", giftRecord.PersonName).First(&card)
		if card.ID == 0 {
			card.PersonName = giftRecord.PersonName
			if err := global.DB.Create(&card).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "创建卡片失败"})
				return
			}
		}
		giftRecord.CardID = card.ID
	}

	// 验证关联的礼薄是否存在
	var giftbook models.GiftBook

	if err := global.DB.First(&giftbook, giftRecord.GiftBookID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "关联的礼薄不存在",
		})
		return
	}
	// 验证当前记录是否属于当前礼薄
	rID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "礼薄 ID 格式错误"})
		return
	}
	if giftRecord.GiftBookID != uint(rID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "记录不属于该礼薄"})
		return
	}
	// 创建礼金记录
	if err := global.DB.Create(&giftRecord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建礼金记录失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":     "礼金记录添加成功",
		"gift_record": giftRecord,
	})

}

// 获取指定礼薄的礼金记录列表
func GetGiftRecordsbyID(c *gin.Context) {
	giftBookID := c.Param("id")
	var giftRecords []models.GiftRecord
	// 验证礼薄是否存在
	var giftbook models.GiftBook
	if err := global.DB.First(&giftbook, giftBookID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "礼薄不存在",
		})
		return
	}
	if err := global.DB.Where("gift_book_id = ?", giftBookID).Find(&giftRecords).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取礼金记录失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"gift_records": giftRecords,
	})

}

// 删除礼金记录
func DeleteGift(c *gin.Context) {
	giftID := c.Param("rid")
	// 先检查记录是否存在
	var giftRecord models.GiftRecord
	if err := global.DB.First(&giftRecord, giftID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "礼金记录不存在",
		})
		return
	}
	//验证当前记录是否属于当前礼薄

	rID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "礼薄 ID 格式错误"})
		return
	}

	// 现在 gbID 是 uint64，比较时转成 uint
	if giftRecord.GiftBookID != uint(rID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "记录不属于该礼薄"})
		return
	}
	if err := global.DB.Delete(&models.GiftRecord{}, giftID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除礼金记录失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "礼金记录删除成功",
	})
}

// 更新礼金记录
func UpdateGift(c *gin.Context) {
	giftID := c.Param("rid")
	var giftRecord models.GiftRecord
	if err := c.ShouldBindJSON(&giftRecord); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	// 先检查记录是否存在
	var exit models.GiftRecord
	if err := global.DB.First(&exit, giftID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "礼金记录不存在",
		})
		return
	}
	//验证当前记录是否属于当前礼薄
	rID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "礼薄 ID 格式错误"})
		return
	}
	if exit.GiftBookID != uint(rID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "记录不属于该礼薄"})
		return
	}
	update := map[string]interface{}{
		"person_name": giftRecord.PersonName,
		"amount":      giftRecord.Amount,
		"address":     giftRecord.Address,
		"gift_note":   giftRecord.GiftNote,
		"card_id":     giftRecord.CardID,
	}
	if err := global.DB.Model(&models.GiftRecord{}).Where("id = ?", giftID).Updates(update).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "更新礼金记录失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "礼金记录更新成功",
	})
}
