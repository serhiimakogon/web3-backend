package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"web3-backend/app/api"
	"web3-backend/app/eth"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	store := api.NewProposalStore()

	srv := api.NewServer(cfg.ListenPort, store)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	listener, err := eth.NewDAOEventListener(cfg.RPCURL(), cfg.DaoContractAddress, cfg.DaoABIPath, store, log.Default())
	if err != nil {
		log.Fatalf("failed to initialize DAO event listener: %v", err)
	}

	go func() {
		if err := listener.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("dao listener exited: %v", err)
		}
	}()

	log.Printf("starting HTTP server on %s", srv.Addr())

	if err := srv.Start(ctx); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}

	log.Println("server shut down gracefully")
}
