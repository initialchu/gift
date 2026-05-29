package config

import (
	"giftmemo/global"
	"giftmemo/models"
	"giftmemo/utils"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func initDB() {
	dsn := AppConfig.Database.Dsn
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)

	}
	//其他配置
	sqlDB, err := db.DB()
	// 设置连接池中空闲连接的最大数量
	sqlDB.SetMaxIdleConns(AppConfig.Database.MaxIdleConns)
	// 设置连接池中打开连接的最大数量
	sqlDB.SetMaxOpenConns(AppConfig.Database.MaxOpenConns)
	//设置连接的最大空闲时间
	sqlDB.SetConnMaxIdleTime(time.Hour)
	if err != nil {
		log.Fatalf("获取数据库连接失败: %v", err)
	}
	// 将 db 赋值给全局变量，供其他包使用
	global.DB = db
	// 自动迁移数据库表结构
	var req models.User
	if err := global.DB.AutoMigrate(&req); err != nil {

		log.Fatalf("自动迁移数据库失败: %v", err)
	}
}

// 暴露一个函数创建管理员
func CreateAdmin() {
	var count int64
	global.DB.Model(&models.User{}).Where("role = ?", "admin").Count(&count)
	if count == 0 {
		hashedPwd, err := utils.HashPassword(AppConfig.Admin.Password)
		if err != nil {
			log.Fatalf("创建管理员失败: %v", err)
		}
		admin := models.User{
			Username: AppConfig.Admin.Name,
			Password: hashedPwd,
			Role:     "admin",
		}
		if err := global.DB.Create(&admin).Error; err != nil {
			log.Fatalf("创建管理员失败: %v", err)
		}
		log.Printf("管理员账号已创建，用户名: %s", AppConfig.Admin.Name)
	}

}
