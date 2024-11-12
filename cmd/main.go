package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwise1/waze_kibris/config"
	deps "github.com/bwise1/waze_kibris/internal/debs"
	api "github.com/bwise1/waze_kibris/internal/http/rest"
)

const (
	allowConnectionsAfterShutdown = 5 * time.Second
)

func main() {
	cfg := config.New()
	deps := deps.New(cfg)

	a := &api.API{
		Config: cfg,
		Deps:   deps,
	}
	go func() {
		log.Printf("Server running on port %v ...", cfg.Port)
		log.Fatal(a.Serve())
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-stopChan

	log.Println("Request to shutdown server. Doing nothing for ", allowConnectionsAfterShutdown)
	waitTimer := time.NewTimer(allowConnectionsAfterShutdown)
	<-waitTimer.C

	log.Println("Shutting down server...")
	//logger.Log.Sugar().Fatal(a.Deps.DAL.DB.Close())
	log.Fatal(a.Shutdown())
}
