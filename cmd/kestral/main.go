package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tnguyen21/kestral-tui/internal/config"
	"github.com/tnguyen21/kestral-tui/internal/server"
)

func main() {
	configPath := flag.String("config", config.DefaultConfigPath, "path to config file")
	port := flag.Int("port", 0, "override listen port")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	if *port > 0 {
		cfg.Port = *port
	}

	srv, err := server.New(&cfg)
	if err != nil {
		log.Fatalf("creating server: %v", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Kestral listening on :%d", cfg.Port)
		if err := srv.Start(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	sig := <-done
	fmt.Println()
	log.Printf("Received %s, shutting down...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
	log.Println("Kestral stopped")
}
