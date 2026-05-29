package config

import (
	"giftmemo/utils"
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

	Admin struct {
		Name     string
		Password string
	}

	Jwt struct {
		Secret      string
		ExpireHours int
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
	//合并本地配置（config.local.yml 不存在也不算错误）
	viper.SetConfigName("config.local")
	if err := viper.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("警告: 加载 config.local.yml 失败: %v", err)
		}
	}
	AppConfig = &Config{}
	//将配置文件中的内容映射到结构体
	if err := viper.Unmarshal(AppConfig); err != nil {
		log.Fatalf("映射配置文件失败: %v", err)
	}
	//将 JWT 配置注入 utils 包（消除硬编码）
	utils.SetJWTConfig(AppConfig.Jwt.Secret, AppConfig.Jwt.ExpireHours)
	initDB()
}
