package controllers

import (
	"fmt"
	"giftmemo/global"
	"giftmemo/models"
	"giftmemo/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 此文件实现登录功能
func GetCaptcha(c *gin.Context) {
	id, b64s, _, err := utils.GenerateCaptcha()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":          id,
		"captcha_img": b64s,
	})
}
func Login(c *gin.Context) {
	var luser models.LoginRequest
	var user models.User
	if err := c.ShouldBindJSON(&luser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//验证验证码
	if !utils.VerifyCaptcha(luser.CaptchaID, luser.CaptchaAnswer) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
		return
	}
	//检查是否被锁定
	if utils.IsLocked(luser.Username) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": fmt.Sprintf("账户已被锁定，请%d分钟后再试", int(utils.LockRemainingTime(luser.Username).Minutes()))})
		return
	}
	//查询数据库验证用户名
	if err := global.DB.Where("username = ?", luser.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户名或密码错误",
		})
		utils.RecordFailedAttempt(luser.Username)
		return

	}
	//验证密码
	if err := utils.CheckPWD(luser.Password, user.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户名或密码错误",
		})
		utils.RecordFailedAttempt(luser.Username)
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
	//登录成功，清除失败记录
	utils.ResetLoginStatus(luser.Username)
	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}
