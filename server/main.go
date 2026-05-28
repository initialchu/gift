package main

import (
	"fmt"
	"giftmemo/config"
	"giftmemo/router"
)

func main() {
	config.InitConfig()

	r := router.SetupRouter()

	r.Run(config.AppConfig.App.Port)
	fmt.Println("服务器已启动，监听端口:", config.AppConfig.App.Port)
}
