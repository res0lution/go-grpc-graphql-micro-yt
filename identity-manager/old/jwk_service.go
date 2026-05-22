package service

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/sirupsen/logrus"
)

type JWKSManager struct {
	cache      jwk.Set
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
	lastUpdate time.Time
	jwksURL    string
	logger     *logrus.Entry
}

// GetKeyByID returns a key from JWKS by kid
func (m *JWKSManager) GetKeyByID(ctx context.Context, kid string) (jwk.Key, error) {
	set, err := m.refreshJWKS(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %w", err)
	}

	key, found := set.LookupKeyID(kid)
	if !found {
		m.logger.WithField("kid", kid).Warn("Key not found in JWKS")
		return nil, fmt.Errorf("key with kid %s not found in JWKS", kid)
	}

	return key, nil
}

// refreshJWKS fetches JWKS from remote or cache
func (m *JWKSManager) refreshJWKS(ctx context.Context) (jwk.Set, error) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	// return cache if still valid
	if m.cache != nil && time.Since(m.lastUpdate) < m.cacheTTL {
		return m.cache, nil
	}

	m.logger.WithField("jwks_url", m.jwksURL).Info("Fetching JWKS from IDP")

	req, err := http.NewRequestWithContext(ctx, "GET", m.jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWKS request: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		m.logger.WithError(err).Error("Failed to fetch JWKS")

		if m.cache != nil {
			m.logger.Warn("Using stale JWKS cache due to fetch error")
			return m.cache, nil
		}

		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.logger.WithField("status_code", resp.StatusCode).Error("JWKS endpoint returned non-200")

		if m.cache != nil {
			m.logger.Warn("Using stale JWKS cache due to HTTP error")
			return m.cache, nil
		}

		return nil, fmt.Errorf("jwks endpoint returned status %d", resp.StatusCode)
	}

	set, err := jwk.ParseReader(resp.Body)
	if err != nil {
		m.logger.WithError(err).Error("Failed to parse JWKS")

		if m.cache != nil {
			m.logger.Warn("Using stale JWKS cache due to parse error")
			return m.cache, nil
		}

		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	m.cache = set
	m.lastUpdate = time.Now()

	m.logger.WithFields(logrus.Fields{
		"keys_count": set.Len(),
		"cache_ttl":  m.cacheTTL,
	}).Info("JWKS cache updated successfully")

	return set, nil
}

// SetCacheTTL updates cache TTL
func (m *JWKSManager) SetCacheTTL(ttl time.Duration) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	m.cacheTTL = ttl
	m.logger.WithField("cache_ttl", ttl).Info("JWKS cache TTL updated")
}

// GetCacheStats returns cache info
func (m *JWKSManager) GetCacheStats() map[string]interface{} {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	stats := map[string]interface{}{
		"cache_exists":    m.cache != nil,
		"last_update":     m.lastUpdate.Format(time.RFC3339),
		"cache_age_hours": time.Since(m.lastUpdate).Hours(),
		"cache_ttl_hours": m.cacheTTL.Hours(),
	}

	if m.cache != nil {
		stats["keys_count"] = m.cache.Len()
	}

	return stats
}
