package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PROYEK3-ULBI/sims-backend/internal/config"
	"github.com/PROYEK3-ULBI/sims-backend/internal/database"
	"github.com/PROYEK3-ULBI/sims-backend/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Apply pending migrations on startup using the session-pooler URL.
	if err := database.RunMigrations(cfg.DatabaseMigrationURL, "migrations"); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	log.Println("[startup] migrations applied")

	// Connect using the transaction-pooler URL for runtime queries.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()
	log.Println("[startup] database connected")

	app := server.New(cfg, pool)

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		addr := ":" + cfg.AppPort
		log.Printf("[startup] listening on %s (env=%s)", addr, cfg.AppEnv)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("[shutdown] signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("[shutdown] complete")
}
