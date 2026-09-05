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

	"github.com/LuisGaravaso/media-indexer/internal/analyzer"
	"github.com/LuisGaravaso/media-indexer/internal/config"
	"github.com/LuisGaravaso/media-indexer/internal/database"
	"github.com/LuisGaravaso/media-indexer/internal/embeddings"
	"github.com/LuisGaravaso/media-indexer/internal/events"
	"github.com/LuisGaravaso/media-indexer/internal/middleware"
	"github.com/LuisGaravaso/media-indexer/internal/pipeline"
	"github.com/LuisGaravaso/media-indexer/internal/repository"
	"github.com/LuisGaravaso/media-indexer/internal/search"
	"github.com/LuisGaravaso/media-indexer/internal/server"
)

func main() {
	cfg := config.Load()

	log.Printf("Starting media-indexer on port %s (GIN_MODE=%s, GCP_PROJECT=%s)...", cfg.Port, cfg.GinMode, cfg.GCPProjectID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Initialize PostgreSQL Pool
	dbPool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("[Warning] Database pool initialization failed: %v", err)
	} else {
		defer dbPool.Close()
	}

	// 2. Initialize Media Analyzer & Text Embedder (Google AI Studio with GEMINI_API_KEY)
	var mediaAnalyzer analyzer.MediaAnalyzer
	var textEmbedder embeddings.Embedder

	if cfg.GeminiAPIKey != "" {
		log.Println("[AI Provider] Initializing Google AI Studio provider with GEMINI_API_KEY")
		aiStudioAnalyzer, err := analyzer.NewAIStudioAnalyzer(ctx, cfg.GeminiAPIKey, cfg.GeminiModel)
		if err != nil {
			log.Printf("[Warning] Google AI Studio analyzer initialization failed: %v", err)
		} else {
			mediaAnalyzer = aiStudioAnalyzer
			defer func() { _ = aiStudioAnalyzer.Close() }()
		}

		aiStudioEmbedder, err := embeddings.NewAIStudioEmbedder(ctx, cfg.GeminiAPIKey, cfg.EmbeddingModel)
		if err != nil {
			log.Printf("[Warning] Google AI Studio embedder initialization failed: %v", err)
		} else {
			textEmbedder = aiStudioEmbedder
		}
	} else {
		log.Println("[Warning] GEMINI_API_KEY is not set. Multimodal indexing and semantic search will remain disabled to prevent Vertex AI billing.")
	}

	// 3. Repositories and Services
	var semanticsRepo repository.SemanticsRepository
	var searchService search.Service
	if dbPool != nil {
		semanticsRepo = repository.NewPostgresSemanticsRepository(dbPool)
		if textEmbedder != nil {
			searchService, _ = search.NewSemanticSearchService(textEmbedder, semanticsRepo)
		}
	}

	// 4. Indexing Pipeline & Worker Pool
	var workerPool *events.WorkerPool
	var subscriber events.Subscriber
	if semanticsRepo != nil && mediaAnalyzer != nil && textEmbedder != nil {
		indexPipeline, err := pipeline.NewMediaIndexingPipeline(cfg.StorageBucket, mediaAnalyzer, textEmbedder, semanticsRepo)
		if err == nil {
			workerPool, _ = events.NewWorkerPool(cfg.WorkerConcurrency, cfg.WorkerConcurrency*3, indexPipeline)
			if workerPool != nil {
				workerPool.Start()
				sub, err := events.NewGCPPubSubSubscriber(ctx, cfg.GCPProjectID, cfg.PubSubSubscription, workerPool)
				if err != nil {
					log.Printf("[Warning] GCP Pub/Sub subscriber initialization failed: %v", err)
				} else {
					subscriber = sub
					go func() {
						if err := subscriber.Start(ctx); err != nil {
							log.Printf("[PubSubSubscriber] Run error: %v", err)
						}
					}()
				}
			}
		}
	}

	// 5. HTTP Router & Auth
	authValidator := middleware.NewJWKSValidator(cfg.SupabaseURL, cfg.SupabaseJWTSecret)
	router := server.SetupRouter(cfg, dbPool, searchService, authValidator)

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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if subscriber != nil {
		_ = subscriber.Stop()
	}
	if workerPool != nil {
		_ = workerPool.Stop(shutdownCtx)
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced shutdown: %v", err)
	}

	log.Println("media-indexer server exited cleanly.")
}
