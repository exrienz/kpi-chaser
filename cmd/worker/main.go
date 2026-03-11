package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/example/kpi-chaser/internal/ai"
	"github.com/example/kpi-chaser/internal/config"
	"github.com/example/kpi-chaser/internal/jobs"
	"github.com/example/kpi-chaser/internal/storage"
)

func main() {
	cfg := config.Load()

	db, err := storage.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := storage.Migrate(db); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	queue := jobs.NewQueue(db)
	provider := ai.NewProvider(cfg)
	worker := jobs.NewWorker(db, queue, provider)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("worker started with concurrency=%d", cfg.WorkerConcurrency)
	if err := worker.Run(ctx, cfg.WorkerConcurrency); err != nil {
		log.Fatalf("worker failed: %v", err)
	}
}
