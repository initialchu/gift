package global

import "gorm.io/gorm"

//全局变量，供其他包使用
var (
	//数据库连接对象，通过此对象可以进行数据库操作
	DB *gorm.DB
)
