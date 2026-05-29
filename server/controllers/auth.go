package controllers

import (
	"giftmemo/global"
	"giftmemo/models"
	"giftmemo/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 此文件实现登录功能
func Login(c *gin.Context) {
	var luser models.LoginRequest
	var user models.User
	if err := c.ShouldBindJSON(&luser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//查询数据库验证用户名
	if err := global.DB.Where("username = ?", luser.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户名或密码错误",
		})
		return
	}
	//验证密码
	if err := utils.CheckPWD(luser.Password, user.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户名或密码错误",
		})
		return
	}

	//调用JWT生成函数
	token, err := utils.GenerateJWT(user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}
