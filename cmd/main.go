package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwise1/waze_kibris/config"
	"github.com/bwise1/waze_kibris/internal/db"
	deps "github.com/bwise1/waze_kibris/internal/debs"
	"github.com/bwise1/waze_kibris/internal/firebaseapp"
	googlemaps "github.com/bwise1/waze_kibris/internal/http/google"
	"github.com/bwise1/waze_kibris/internal/http/mapbox"
	api "github.com/bwise1/waze_kibris/internal/http/rest"
	stadiamaps "github.com/bwise1/waze_kibris/internal/http/stadia_maps"

	"github.com/bwise1/waze_kibris/internal/http/valhalla"
	smtp "github.com/bwise1/waze_kibris/util/email"
)

const (
	allowConnectionsAfterShutdown = 1 * time.Second
)

func main() {
	cfg := config.New()
	deps := deps.New(cfg)

	mailer := smtp.NewMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)

	database, err := db.New(cfg.Dsn)
	if err != nil {
		log.Panicln("failed to connect to database", "error", err)
	}
	valhallaClient := valhalla.NewValhallaClient(cfg.ValhallaURL)
	log.Printf("Valhalla client initialized with BaseURL: %s", cfg.ValhallaURL)

	stadiaClient := stadiamaps.NewClient(cfg.StadiaMapsAPIKey)
	log.Printf("stadia client initialized")

	googleMapsClient := googlemaps.NewGoogleMapsClient(cfg.GoogleMapsAPIKey)
	mapboxClient := mapbox.NewMapboxClient(cfg.MapboxAPIKey)
	log.Printf("Mapbox client initialized")

	fbAuth, fbMessaging, err := firebaseapp.InitAuthAndMessaging(context.Background(), cfg.FirebaseCredentialsPath)
	if err != nil {
		log.Panicln("failed to init Firebase", err)
	}
	if fbAuth != nil {
		log.Println("Firebase Auth client initialized (ID token verification enabled)")
		if fbMessaging != nil {
			log.Println("Firebase Cloud Messaging client initialized (push send enabled)")
		} else {
			log.Println("Firebase Messaging unavailable (FCM send disabled)")
		}
	} else {
		log.Println("Firebase not configured (set FIREBASE_CREDENTIALS_PATH or GOOGLE_APPLICATION_CREDENTIALS)")
	}

	a := &api.API{
		Config:             cfg,
		Deps:               deps,
		Mailer:             mailer,
		DB:                 database.Pool(),
		ValhallaClient:     valhallaClient,
		StadiaClient:       stadiaClient,
		GoogleMapsClient:   googleMapsClient,
		MapboxClient:       mapboxClient,
		FirebaseAuth:       fbAuth,
		FirebaseMessaging:  fbMessaging,
	}
	a.Init()
	go deps.WebSocket.Run()
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

	database.Close()
	log.Fatal("Database connections closed.")

	log.Fatal(a.Shutdown())
}
