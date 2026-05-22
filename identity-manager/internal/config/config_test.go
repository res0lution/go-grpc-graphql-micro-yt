package config

import "testing"

func TestLoad_RequiresDBUser(t *testing.T) {
	t.Setenv("DB_USER", "")
	if _, err := Load(); err == nil {
		t.Fatalf("expected DB_USER required error")
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DB_USER", "postgres")
	t.Setenv("APP_PORT", "")
	t.Setenv("SESSION_COOKIE_SECURE", "")
	t.Setenv("SESSION_COOKIE_HTTP_ONLY", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfg.App.Port != "8088" {
		t.Fatalf("expected default APP_PORT=8088, got %s", cfg.App.Port)
	}
	if !cfg.Cookie.Secure || !cfg.Cookie.HTTPOnly {
		t.Fatalf("expected secure+httponly defaults")
	}
}

func TestLoad_ProductionValidation(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("GIN_MODE", "release")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("IDP_HOST", "https://idp.example.com")
	t.Setenv("IDP_CLIENT_ID", "client-id")
	t.Setenv("IDP_CLIENT_SECRET", "client-secret")
	t.Setenv("IDP_REDIRECT_URI", "https://app.example.com/cb")
	t.Setenv("CORE_INTERNAL_AUTH_TOKEN", "internal-token")

	if _, err := Load(); err != nil {
		t.Fatalf("unexpected production validation error: %v", err)
	}
}

func TestLoad_ProductionValidationRejectsInsecure(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("DB_SSLMODE", "disable")

	if _, err := Load(); err == nil {
		t.Fatalf("expected production validation to fail")
	}
}
