package main

import (
	"os"
	"wsim/pkg/postgresql"
	"wsim/user/routes"
	wsutils "wsim/utils"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func main() {
	postgresql.InitPostgreSQL()
	logger := wsutils.SetupLogger(hlog.LevelDebug)
	hlog.SetLogger(logger)

	port := os.Getenv("PORT")
	h := server.New(server.WithHostPorts("0.0.0.0:" + port))

	routes.InitRouter(h)

	h.Spin()
}
