package main

import (
	"flag"

	"github.com/mingkaic/accretion/api"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func init() {
	var lvl string

	flag.StringVar(&lvl, "log_level", "debug", "Log level")
	flag.Parse()

	log_level, err := log.ParseLevel(lvl)
	if nil != err {
		panic(err)
	}
	log.SetLevel(log_level)
}

func main() {
	addr := "localhost:8069"
	log.Infof("Serving on %s", addr)

	var opts []grpc.ServerOption
	opts = append(opts, grpc.MaxRecvMsgSize(1024*1024*32)) // 32MB
	app := api.NewAccretionAPI()
	if err := app.Run(addr, opts); err != nil {
		log.Fatal(err)
	}
}
