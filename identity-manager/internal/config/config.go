package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	IDP      IDPConfig
	Core     CoreConfig
	Cookie   CookieConfig
}

type AppConfig struct {
	Env         string
	LogLevel    string
	Port        string
	GinMode     string
	FrontendURL string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type IDPConfig struct {
	Host         string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	JWKSURL      string
}

type CoreConfig struct {
	BaseURL           string
	InternalAuthToken string
}

type CookieConfig struct {
	Name     string
	Secure   bool
	HTTPOnly bool
	SameSite string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Env:         getEnv("APP_ENV", "development"),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
			Port:        getEnv("APP_PORT", "8088"),
			GinMode:     getEnv("GIN_MODE", "debug"),
			FrontendURL: getEnv("PORTAL_UI_HOST", "/"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     getEnv("DB_NAME", "identity_manager"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		IDP: IDPConfig{
			Host:         getEnv("IDP_HOST", ""),
			ClientID:     getEnv("IDP_CLIENT_ID", ""),
			ClientSecret: getEnv("IDP_CLIENT_SECRET", ""),
			RedirectURI:  getEnv("IDP_REDIRECT_URI", ""),
			JWKSURL:      getEnv("IDP_JWKS_URL", ""),
		},
		Core: CoreConfig{
			BaseURL:           getEnv("CORE_BACKEND_URL", ""),
			InternalAuthToken: getEnv("CORE_INTERNAL_AUTH_TOKEN", ""),
		},
		Cookie: CookieConfig{
			Name:     getEnv("SESSION_COOKIE_NAME", "session_id"),
			Secure:   getEnvAsBool("SESSION_COOKIE_SECURE", true),
			HTTPOnly: getEnvAsBool("SESSION_COOKIE_HTTP_ONLY", true),
			SameSite: getEnv("SESSION_COOKIE_SAMESITE", "Lax"),
		},
	}

	if cfg.Database.User == "" {
		return nil, fmt.Errorf("DB_USER is required")
	}
	if err := validateProductionConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validateProductionConfig(cfg *Config) error {
	if strings.ToLower(strings.TrimSpace(cfg.App.Env)) != "production" {
		return nil
	}

	if strings.TrimSpace(cfg.App.GinMode) != "release" {
		return fmt.Errorf("GIN_MODE must be release in production")
	}
	if strings.EqualFold(strings.TrimSpace(cfg.Database.SSLMode), "disable") {
		return fmt.Errorf("DB_SSLMODE must not be disable in production")
	}

	required := map[string]string{
		"IDP_HOST":                cfg.IDP.Host,
		"IDP_CLIENT_ID":           cfg.IDP.ClientID,
		"IDP_CLIENT_SECRET":       cfg.IDP.ClientSecret,
		"IDP_REDIRECT_URI":        cfg.IDP.RedirectURI,
		"CORE_INTERNAL_AUTH_TOKEN": cfg.Core.InternalAuthToken,
	}
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required in production", key)
		}
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
