package main

import (
	"log"
	"net/http"

	"github.com/example/kpi-chaser/internal/config"
	"github.com/example/kpi-chaser/internal/httpapi"
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

	server, err := httpapi.NewServer(cfg, db)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	log.Printf("api server listening on %s", cfg.HTTPAddress)
	if err := http.ListenAndServe(cfg.HTTPAddress, server.Router()); err != nil {
		log.Fatalf("serve http: %v", err)
	}
}
