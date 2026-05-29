package controllers

import (
	"giftmemo/global"
	"giftmemo/models"
	"giftmemo/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

//此文件时对用户的增删改查功能的实现

// 创建用户
func CreateUser(c *gin.Context) {
	var req models.LoginRequest
	// 绑定JSON请求体到结构体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//检查用户名是否已存在
	var euser models.User
	if result := global.DB.Where("username = ?", req.Username).First(&euser); result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}
	//hash密码
	hashedPwd, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	//构造用户对象并保存到数据库
	user := models.User{
		Username: req.Username,
		Password: hashedPwd,
	}

	if err := global.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	//返回成功响应
	c.JSON(http.StatusCreated, gin.H{"message": "用户创建成功", "username": user.Username})
}
