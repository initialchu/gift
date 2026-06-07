package router

import (
	"giftmemo/controllers"
	"giftmemo/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	auth := r.Group("/api/auth")
	{
		auth.POST("/login", controllers.Login)
		auth.GET("/captcha", controllers.GetCaptcha)
	}
	api := r.Group("/api")
	api.Use(middlewares.AuthMiddleware())
	{
		// 获取礼薄列表接口
		api.GET("/giftbooks", controllers.GetgiftBooks)
		// 获取单个礼薄详情接口
		api.GET("/giftbook/:id", controllers.GetgiftBookbyID)
		//获取指定礼薄的礼金记录列表接口
		api.GET("/giftrecord/:id/records", controllers.GetGiftRecordsbyID)
		// 获取人情卡片汇总列表
		api.GET("/cards", controllers.GetCards)
		// 获取某张卡片的往来明细
		api.GET("/cards/detail", controllers.GetCardDetail)
	}

	admin := r.Group("/api/admin")
	admin.Use(middlewares.AuthMiddleware())
	admin.Use(middlewares.AdminMiddleware())

	{ //创建用户接口
		admin.POST("/create", controllers.CreateUser)
		// 获取用户列表接口

		//创建礼薄接口
		admin.POST("/giftbook", controllers.CreateGiftBook)

		//编辑礼薄接口
		admin.POST("/giftbook/edit/:id", controllers.UpdateGiftBook)
		//删除礼薄接口
		admin.POST("/giftbook/:id", controllers.DeleteGiftBook)
		//添加礼金记录接口
		admin.POST("/giftrecord/:id/records", controllers.AddGift)

		//删除礼金记录接口
		admin.POST("/giftrecord/:id/records/:rid", controllers.DeleteGift)
		//编辑礼金记录接口
		admin.POST("/giftrecord/:id/records/:rid/edit", controllers.UpdateGift)

	}
	return r
}
