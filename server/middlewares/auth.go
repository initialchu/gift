package middlewares

import (
	"giftmemo/utils"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 在这里实现认证逻辑，检查JWT token
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(401, gin.H{"error": "未提供token"})
			c.Abort()
			return
		}
		// 验证token的有效性
		username, role, err := utils.ParseJWT(token)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的token"})
			c.Abort()
			return
		}
		// 将用户名存储在上下文中，以便后续处理使用
		//可以跨中间件传递数据
		c.Set("username", username)
		c.Set("role", role)
		c.Next()
	}
}

// 管理员权限中间件
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			c.JSON(403, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}
