package repository

import (
	"errors"
	"strings"
	"testing"
	"time"
)

type mockUserScanner struct {
	err        error
	groupsJSON []byte
}

func (m *mockUserScanner) Scan(dest ...any) error {
	if m.err != nil {
		return m.err
	}

	*(dest[0].(*string)) = "u1"
	*(dest[1].(*string)) = "id-1"
	*(dest[2].(*string)) = "sub-1"
	*(dest[3].(*string)) = "alice"
	*(dest[4].(*string)) = "alice@example.com"
	*(dest[5].(*string)) = "100"
	*(dest[6].(*string)) = "alice-win"
	*(dest[7].(*string)) = "Alice"
	*(dest[8].(*string)) = "Doe"
	*(dest[9].(*string)) = "Alice Doe"
	*(dest[10].(*[]byte)) = m.groupsJSON
	*(dest[11].(*bool)) = true
	now := time.Now()
	*(dest[12].(*time.Time)) = now
	*(dest[13].(*time.Time)) = now
	*(dest[14].(*time.Time)) = now

	return nil
}

func TestDecodeUserGroups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		wantLen int
		wantErr bool
	}{
		{name: "empty", input: nil, wantLen: 0},
		{name: "valid", input: []byte(`["A","B"]`), wantLen: 2},
		{name: "invalid", input: []byte(`{bad`), wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			groups, err := decodeUserGroups(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(groups) != tt.wantLen {
				t.Fatalf("expected len %d, got %d", tt.wantLen, len(groups))
			}
		})
	}
}

func TestScanUser(t *testing.T) {
	t.Parallel()

	t.Run("scan error", func(t *testing.T) {
		t.Parallel()
		_, err := scanUser(&mockUserScanner{err: errors.New("scan failed")})
		if err == nil {
			t.Fatal("expected scan error")
		}
	})

	t.Run("invalid groups json", func(t *testing.T) {
		t.Parallel()
		_, err := scanUser(&mockUserScanner{groupsJSON: []byte("{bad")})
		if err == nil {
			t.Fatal("expected decode error")
		}
		if !strings.Contains(err.Error(), "failed to unmarshal groups") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		user, err := scanUser(&mockUserScanner{groupsJSON: []byte(`["A","B"]`)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.ID == "" {
			t.Fatal("expected user id")
		}
		if len(user.Groups) != 2 {
			t.Fatalf("expected 2 groups, got %d", len(user.Groups))
		}
	})
}
