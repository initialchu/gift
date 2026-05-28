package config

import (
	"giftmemo/global"
	"giftmemo/models"
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
