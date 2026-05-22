package service

import (
	"sync"
	"time"
)

type authStateEntry struct {
	nonce     string
	expiresAt time.Time
}

type authStateStore struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]authStateEntry
}

func newAuthStateStore(ttl time.Duration) *authStateStore {
	return &authStateStore{
		ttl:     ttl,
		entries: make(map[string]authStateEntry),
	}
}

func (s *authStateStore) Put(state, nonce string) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked(now)
	s.entries[state] = authStateEntry{
		nonce:     nonce,
		expiresAt: now.Add(s.ttl),
	}
}

func (s *authStateStore) Consume(state string) (string, bool) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked(now)
	entry, ok := s.entries[state]
	if !ok {
		return "", false
	}

	delete(s.entries, state)
	return entry.nonce, true
}

func (s *authStateStore) cleanupLocked(now time.Time) {
	for key, entry := range s.entries {
		if now.After(entry.expiresAt) {
			delete(s.entries, key)
		}
	}
}
