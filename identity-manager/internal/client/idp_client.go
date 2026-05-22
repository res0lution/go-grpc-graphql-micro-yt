package client

import "identity-manager/internal/config"

type IDPClient struct {
	cfg config.IDPConfig
}

func NewIDPClient(cfg config.IDPConfig) *IDPClient {
	return &IDPClient{cfg: cfg}
}
