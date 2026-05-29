package main

import (
	"fmt"
	"giftmemo/config"
	"giftmemo/router"
)

func main() {
	config.InitConfig()

	// 初始化种子数据：首次启动自动创建管理员
	config.CreateAdmin()

	r := router.SetupRouter()

	fmt.Println("服务器已启动，监听端口:", config.AppConfig.App.Port)
	r.Run(config.AppConfig.App.Port)

}
