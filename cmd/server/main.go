package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/guilferri/lumo-api/internal/api"
	"github.com/guilferri/lumo-api/internal/browser"
)

func main() {
	// -------------------------------------------------------------
	// Initialise the Playwright driver (single shared browser).
	// -------------------------------------------------------------
	drv, err := browser.NewDriver()
	if err != nil {
		log.Fatalf("🚨 Failed to start browser driver: %v", err)
	}
	defer func() {
		_ = drv.Close()
	}()

	// -------------------------------------------------------------
	// HTTP server – listen on configurable port (default 8080).
	// -------------------------------------------------------------
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: api.NewHandler(drv),
	}

	// Graceful shutdown handling.
	go func() {
		log.Printf("🟢 Lumo API listening on http://0.0.0.0:%s/v1/prompt", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("🚨 ListenAndServe: %v", err)
		}
	}()

	// Wait for SIGINT/SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("🛑 Shutting down…")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("🚨 Shutdown error: %v", err)
	}
	log.Println("✅ Bye!")
}
