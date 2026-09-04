package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LuisGaravaso/media-indexer/internal/config"
	"github.com/LuisGaravaso/media-indexer/internal/server"
)

func main() {
	cfg := config.Load()

	log.Printf("Starting media-indexer on port %s (GIN_MODE=%s, GCP_PROJECT=%s)...", cfg.Port, cfg.GinMode, cfg.GCPProjectID)

	router := server.SetupRouter(cfg, nil)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server listen failed: %v", err)
		}
	}()

	log.Printf("media-indexer is ready to serve traffic on :%s", cfg.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down media-indexer server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced shutdown: %v", err)
	}

	log.Println("media-indexer server exited cleanly.")
}
