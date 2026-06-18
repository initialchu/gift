package controllers

import (
	"giftmemo/global"
	"giftmemo/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 礼薄控制器，处理与礼薄相关的HTTP请求

// 创建礼薄的处理函数
func CreateGiftBook(c *gin.Context) {
	var giftbook models.GiftBook
	// 处理创建礼薄的逻辑
	// 从请求体中绑定JSON数据到giftbook结构体
	if err := c.ShouldBindJSON(&giftbook); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//查找是否有同名的礼薄
	var exit models.GiftBook
	result := global.DB.Where("event_name=?", giftbook.EventName).First(&exit)
	if result.Error == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "同名的礼薄已存在"})
		return
	}
	// 将创建者信息从上下文中获取并设置到giftbook结构体中
	createdby, ok := c.Get("username")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取创建者信息失败"})
		return
	}
	giftbook.CreatedBy = createdby.(string)
	// 将新的礼薄记录保存到数据库中
	if err := global.DB.Create(&giftbook).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":  "礼薄创建成功",
		"giftbook": giftbook,
	})
}

// 获取礼薄列表的处理函数
func GetgiftBooks(c *gin.Context) {
	var giftbooks []models.GiftBook
	//筛选direction参数
	direction := c.Query("direction")
	//如果direction参数不为空，则根据direction筛选礼薄记录，否则返回所有记录
	if direction != "" {
		if direction != "来" && direction != "去" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "direction参数必须是'来'或'去'"})
			return
		}
		// 从数据库中查询所有符合direction条件的礼薄记录
		if err := global.DB.Where("direction=?", direction).Find(&giftbooks).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

	} else {
		// 从数据库中查询所有礼薄记录
		if err := global.DB.Find(&giftbooks).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	// 将查询到的礼薄记录作为JSON响应返回给客户端
	c.JSON(http.StatusOK, gin.H{
		"giftbooks": giftbooks,
	})

}

// 获取单个礼薄详情的处理函数
func GetgiftBookbyID(c *gin.Context) {
	var giftbook models.GiftBook
	id := c.Param("id")
	// 从数据库中查询指定ID的礼薄记录
	if err := global.DB.Preload("Records").Where("id=?", id).First(&giftbook).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"giftbook": giftbook,
	})
}

// 删除礼薄的处理函数
func DeleteGiftBook(c *gin.Context) {
	var giftbook models.GiftBook
	id := c.Param("id")
	// 从数据库中查询指定ID的礼薄记录
	// 先确认礼薄存在
	if err := global.DB.Where("id = ?", id).First(&giftbook).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "礼薄未找到"})
		return
	}
	if err := global.DB.Where("gift_book_id=?", id).Delete(&models.GiftRecord{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 删除指定ID的礼薄记录
	if err := global.DB.Where("id=?", id).Delete(&giftbook).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "礼薄删除成功",
	})
}

// 修改礼薄的处理函数
func UpdateGiftBook(c *gin.Context) {
	var gift models.GiftBook

	if err := c.ShouldBindJSON(&gift); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	// 从数据库中查询指定ID的礼薄记录
	var exit models.GiftBook
	if err := global.DB.Where("id=?", gift.ID).First(&exit).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "礼薄未找到"})
		return
	}
	update := map[string]interface{}{
		"event_name": gift.EventName,
		"event_date": gift.EventDate,
		"direction":  gift.Direction,
	}
	// 更新礼薄信息
	if err := global.DB.Model(&exit).Updates(update).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "礼薄更新成功",
	})

}
