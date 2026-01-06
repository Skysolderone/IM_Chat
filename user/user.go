package main

import (
	"log"
	"os"
	"wsim/pkg/config"
	"wsim/pkg/postgresql"
	"wsim/user/routes"
	wsutils "wsim/utils"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func main() {
	if err := config.LoadConfig(""); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	postgresql.InitPostgreSQL()
	logger := wsutils.SetupLogger(hlog.LevelDebug)
	hlog.SetLogger(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = config.GetServerPort()
	}
	h := server.New(server.WithHostPorts("0.0.0.0:" + port))

	routes.InitRouter(h)

	h.Spin()
}
