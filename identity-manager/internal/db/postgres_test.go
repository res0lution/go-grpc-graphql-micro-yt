package db

import (
	"net/url"
	"strings"
	"testing"

	"identity-manager/internal/config"
)

func TestValidateDatabaseConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     config.DatabaseConfig
		wantErr string
	}{
		{
			name: "valid config",
			cfg: config.DatabaseConfig{
				Host: "localhost",
				Port: "5432",
				User: "postgres",
				Name: "identity_manager",
			},
		},
		{
			name: "missing host",
			cfg: config.DatabaseConfig{
				Port: "5432",
				User: "postgres",
				Name: "identity_manager",
			},
			wantErr: "db host is required",
		},
		{
			name: "missing port",
			cfg: config.DatabaseConfig{
				Host: "localhost",
				User: "postgres",
				Name: "identity_manager",
			},
			wantErr: "db port is required",
		},
		{
			name: "missing user",
			cfg: config.DatabaseConfig{
				Host: "localhost",
				Port: "5432",
				Name: "identity_manager",
			},
			wantErr: "db user is required",
		},
		{
			name: "missing name",
			cfg: config.DatabaseConfig{
				Host: "localhost",
				Port: "5432",
				User: "postgres",
			},
			wantErr: "db name is required",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateDatabaseConfig(tt.cfg)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}

func TestBuildPostgresConnStringEscapesValues(t *testing.T) {
	t.Parallel()

	cfg := config.DatabaseConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "user@corp",
		Password: "p@ss:/?word",
		Name:     "identity_manager",
		SSLMode:  "require",
	}

	connString := buildPostgresConnString(cfg)
	parsedURL, err := url.Parse(connString)
	if err != nil {
		t.Fatalf("failed to parse conn string: %v", err)
	}

	if got := parsedURL.User.Username(); got != cfg.User {
		t.Fatalf("expected user %q, got %q", cfg.User, got)
	}

	password, ok := parsedURL.User.Password()
	if !ok {
		t.Fatal("expected password to be present")
	}
	if password != cfg.Password {
		t.Fatalf("expected password %q, got %q", cfg.Password, password)
	}

	if got := strings.TrimPrefix(parsedURL.Path, "/"); got != cfg.Name {
		t.Fatalf("expected db name %q, got %q", cfg.Name, got)
	}

	if got := parsedURL.Query().Get("sslmode"); got != cfg.SSLMode {
		t.Fatalf("expected sslmode %q, got %q", cfg.SSLMode, got)
	}
}
