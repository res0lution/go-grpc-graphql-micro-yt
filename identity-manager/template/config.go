package template

import (
	"fmt"
	"strings"
	"time"
)

const (
	PrimarySourceIDM    = "idm"
	PrimarySourceLegacy = "legacy"
)

type Config struct {
	BaseURL       string
	InternalToken string

	Enabled         bool
	DualReadEnabled bool
	PrimarySource   string
	FailOpen        bool

	Timeout      time.Duration
	MaxRetries   int
	RetryBackoff time.Duration

	SessionCookieName string
}

func (c *Config) Normalize() {
	if c.Timeout <= 0 {
		c.Timeout = 500 * time.Millisecond
	}
	if c.RetryBackoff <= 0 {
		c.RetryBackoff = 100 * time.Millisecond
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = 0
	}
	if strings.TrimSpace(c.PrimarySource) == "" {
		c.PrimarySource = PrimarySourceLegacy
	}
	if strings.TrimSpace(c.SessionCookieName) == "" {
		c.SessionCookieName = "session_id"
	}
}

func (c Config) Validate() error {
	switch strings.ToLower(strings.TrimSpace(c.PrimarySource)) {
	case PrimarySourceIDM, PrimarySourceLegacy:
	default:
		return fmt.Errorf("unsupported primary source: %q", c.PrimarySource)
	}

	if c.Enabled {
		if strings.TrimSpace(c.BaseURL) == "" {
			return fmt.Errorf("base url is required when idm is enabled")
		}
		if strings.TrimSpace(c.InternalToken) == "" {
			return fmt.Errorf("internal token is required when idm is enabled")
		}
	}

	return nil
}
