package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xuyuanzhang1122/bililive-server-update/internal/config"
	"github.com/xuyuanzhang1122/bililive-server-update/internal/server"
	"github.com/xuyuanzhang1122/bililive-server-update/internal/store"
)

func main() {
	cfg := config.FromEnv()
	fileStore, err := store.NewFileStore(cfg.DataDir)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	handler := server.New(cfg, fileStore)
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("bililive-server-update listening on %s", cfg.Addr)
		errCh <- srv.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("received %s, shutting down", sig)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown failed: %v", err)
	}
}
