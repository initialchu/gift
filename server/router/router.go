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

	}
	api := r.Group("/api")
	api.Use(middlewares.AuthMiddleware())
	{
		api.GET("/profile")

	}
	admin := r.Group("/api/admin")
	admin.Use(middlewares.AuthMiddleware())
	admin.Use(middlewares.AdminMiddleware())

	{
		admin.POST("/create", controllers.CreateUser)
	}
	return r
}
