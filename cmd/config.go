package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	ListenPort         string
	AlchemyAPIKey      string
	AlchemyEndpointURL string
	DaoContractAddress string
	DaoABIPath         string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		ListenPort:         envOrDefault("APP_SERVER_LISTEN_PORT", "8080"),
		AlchemyAPIKey:      envOrDefault("APP_ALCHEMY_API_KEY", ""),
		AlchemyEndpointURL: envOrDefault("APP_ALCHEMY_ENDPOINT_URL", ""),
		DaoContractAddress: envOrDefault("APP_DAO_CONTRACT_ADDRESS", ""),
		DaoABIPath:         envOrDefault("APP_DAO_ABI_PATH", "app/eth/daoABI.json"),
	}

	if cfg.AlchemyAPIKey == "" {
		return nil, fmt.Errorf("APP_ALCHEMY_API_KEY is required")
	}
	if cfg.AlchemyEndpointURL == "" {
		return nil, fmt.Errorf("APP_ALCHEMY_ENDPOINT_URL is required")
	}
	if cfg.DaoContractAddress == "" {
		return nil, fmt.Errorf("APP_DAO_CONTRACT_ADDRESS is required")
	}
	if !common.IsHexAddress(cfg.DaoContractAddress) {
		return nil, fmt.Errorf("APP_DAO_CONTRACT_ADDRESS must be a valid hex address")
	}
	if cfg.DaoABIPath == "" {
		return nil, fmt.Errorf("APP_DAO_ABI_PATH is required")
	}

	return cfg, nil
}

func (c *Config) RPCURL() string {
	endpoint := strings.TrimRight(c.AlchemyEndpointURL, "/")
	return fmt.Sprintf("%s/%s", endpoint, c.AlchemyAPIKey)
}

func envOrDefault(key, value string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return value
}
