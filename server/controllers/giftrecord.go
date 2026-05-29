package controllers

import (
	"giftmemo/global"
	"giftmemo/models"
	"net/http"

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
	// 验证关联的礼薄是否存在
	var giftbook models.GiftBook

	if err := global.DB.First(&giftbook, giftRecord.GiftBookID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "关联的礼薄不存在",
		})
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
