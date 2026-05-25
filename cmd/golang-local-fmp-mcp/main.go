package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang-local-fmp-mcp/internal/config"
	"golang-local-fmp-mcp/internal/server"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "Path to config.yaml")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv := server.New(cfg)
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("server: %v", err)
	}
	log.Printf("bye")
}
