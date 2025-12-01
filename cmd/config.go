package main

import (
	"fmt"
	"os"
)

type Config struct {
	ListenPort         string
	AlchemyApiKey      string
	AlchemyEndpointURL string
	DaoContractAddress string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		ListenPort:         envOrDefault("APP_SERVER_LISTEN_PORT", "8080"),
		AlchemyApiKey:      envOrDefault("APP_ALCHEMY_API_KEY", ""),
		AlchemyEndpointURL: envOrDefault("APP_ALCHEMY_ENDPOINT_URL", ""),
		DaoContractAddress: envOrDefault("APP_DAO_CONTRACT_ADDRESS", ""),
	}

	if cfg.AlchemyApiKey == "" {
		return &Config{}, fmt.Errorf("alchemy api key is required")
	}
	if cfg.AlchemyEndpointURL == "" {
		return &Config{}, fmt.Errorf("alchemy endpoint url is required")
	}
	if cfg.DaoContractAddress == "" {
		return &Config{}, fmt.Errorf("dao contract address is required")
	}

	return cfg, nil
}

func envOrDefault(key, value string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return value
}
