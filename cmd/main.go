package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"web3-backend/app/api"
)

func main() {
	loadEnvFile()

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	srv := api.NewServer(cfg.ListenPort)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("starting HTTP server on 0.0.0.0:%s", cfg.ListenPort)

	if err := srv.Start(ctx); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}

	log.Println("server shut down gracefully")
}

func loadEnvFile() {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		_ = godotenv.Load()
		return
	}

	if err := godotenv.Load(envFile); err != nil {
		log.Printf("warning: failed to load env file %s: %v", envFile, err)
	}
}
