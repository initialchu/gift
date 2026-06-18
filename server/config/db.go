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
	Autotable()
	// 迁移卡片数据
	global.DB.AutoMigrate(&models.Card{})
	global.DB.AutoMigrate(&models.GiftRecord{})
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

func Autotable() {
	var req models.User
	if err := global.DB.AutoMigrate(&req); err != nil {

		log.Fatalf("自动迁移数据库失败: %v", err)
	}
	var giftbooks models.GiftBook
	if err := global.DB.AutoMigrate(&giftbooks); err != nil {
		log.Fatalf("自动迁移数据库失败: %v", err)

	}
	var giftrecords models.GiftRecord
	if err := global.DB.AutoMigrate(&giftrecords); err != nil {
		log.Fatalf("自动迁移数据库失败: %v", err)
	}
	var cards models.Card
	if err := global.DB.AutoMigrate(&cards); err != nil {
		log.Fatalf("自动迁移数据库失败: %v", err)
	}
}

// MigrateCards 将现有 gift_records 按 person_name 归集为卡片，回填 card_id
// card_id 为 0 或 NULL 的记录视为未迁移
func MigrateCards() {
	var count int64
	global.DB.Model(&models.GiftRecord{}).Where("card_id IS NULL OR card_id = 0").Count(&count)
	if count == 0 {
		return
	}
	log.Printf("开始数据迁移：%d 条记录需要关联卡片...", count)

	// ① 去重人名 → 建卡片
	var names []string
	global.DB.Model(&models.GiftRecord{}).
		Select("DISTINCT person_name").
		Where("card_id IS NULL OR card_id = 0").
		Pluck("person_name", &names)

	for _, name := range names {
		global.DB.Create(&models.Card{PersonName: name})
	}
	log.Printf("已创建 %d 张卡片", len(names))

	// ② 回填 card_id：按 person_name 匹配
	global.DB.Exec(`
		UPDATE gift_records
		SET card_id = (SELECT id FROM cards WHERE cards.person_name = gift_records.person_name)
		WHERE card_id IS NULL OR card_id = 0
	`)
	log.Printf("迁移完成")
}
