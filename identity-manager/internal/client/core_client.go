package client

import "identity-manager/internal/config"

type CoreClient struct {
	cfg config.CoreConfig
}

func NewCoreClient(cfg config.CoreConfig) *CoreClient {
	return &CoreClient{cfg: cfg}
}
