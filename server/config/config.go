package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	App struct {
		Name string
		Port string
	}

	Database struct {
		Dsn          string
		MaxIdleConns int
		MaxOpenConns int
	}
}

var AppConfig *Config

func InitConfig() {
	//设置配置文件的名称和类型，路径
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath("./config")
	//读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}
	AppConfig = &Config{}
	//将配置文件中的内容映射到结构体
	if err := viper.Unmarshal(AppConfig); err != nil {
		log.Fatalf("映射配置文件失败: %v", err)
	}
	initDB()
}
