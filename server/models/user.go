package models

import "gorm.io/gorm"

// 用户模型(数据库表结构)
type User struct {
	gorm.Model
	Username string `gorm:"unique;not null" json:"username" binding:"required"`
	Password string `gorm:"not null" json:"-" binding:"required"`
	Role     string `gorm:"type:varchar(16);default:user;not null" json:"role"`
}

// 登录请求结构体
type LoginRequest struct {
	Username      string `json:"username" binding:"required"`
	Password      string `json:"password" binding:"required"`
	CaptchaID     string `json:"captcha_id" binding:"required"`
	CaptchaAnswer string `json:"captcha_answer" binding:"required"`
}
