package main

import (
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

	h := server.New(server.WithHostPorts("0.0.0.0:9091"))

	routes.InitRouter(h)

	h.Spin()
}
