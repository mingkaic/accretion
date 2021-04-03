package main

import (
	"flag"
	"os"

	"github.com/mingkaic/accretion/api"
	log "github.com/sirupsen/logrus"
	"github.com/zenazn/goji/bind"
	"github.com/zenazn/goji/graceful"
	"google.golang.org/grpc"
)

const (
	grpcAddr = "localhost:8069"
	httpAddr = "localhost:8071"
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
	log.Infof("Serving grpc on %s, http on %s", grpcAddr, httpAddr)

	var (
		grpcOpts []grpc.ServerOption
		dialOpts []grpc.DialOption

		count             int
		failed            bool
		gracefullyStopped bool
	)
	grpcOpts = append(grpcOpts, grpc.MaxRecvMsgSize(1024*1024*64)) // 32MB
	dialOpts = append(dialOpts, grpc.WithInsecure())
	app := api.NewAccretionAPI()

	graceful.HandleSignals()
	bind.Ready()
	graceful.PreHook(func() {
		gracefullyStopped = true
		log.Info("Server received signal, gracefully stopping.")
	})
	graceful.PostHook(func() {
		log.Info("Server stopped")
	})

	errs := make(chan error, 2)
	app.Run(httpAddr, grpcAddr, errs, grpcOpts, api.HTTPOpts{DialOpts: dialOpts})

	for err := range errs {
		if err != nil {
			if !gracefullyStopped {
				log.Error(err)
				failed = true
			}
		}
		count++
		if count == cap(errs) {
			close(errs)
			break
		}
	}
	if failed {
		os.Exit(1)
	}
	//graceful.Wait()
}
