package service

import (
	"testing"
	"time"
)

func TestAuthStateStore_ConsumeOnce(t *testing.T) {
	store := newAuthStateStore(time.Minute)
	store.Put("s1", "n1")

	nonce, ok := store.Consume("s1")
	if !ok {
		t.Fatalf("expected state to exist")
	}
	if nonce != "n1" {
		t.Fatalf("expected nonce n1, got %s", nonce)
	}

	if _, ok := store.Consume("s1"); ok {
		t.Fatalf("expected state to be consumed once")
	}
}

func TestAuthStateStore_Expires(t *testing.T) {
	store := newAuthStateStore(5 * time.Millisecond)
	store.Put("s1", "n1")
	time.Sleep(10 * time.Millisecond)

	if _, ok := store.Consume("s1"); ok {
		t.Fatalf("expected state to expire")
	}
}
