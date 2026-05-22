package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAPIError_JSONContract(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(APIError{
		Code:    "UNAUTHORIZED",
		Message: "auth required",
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	payload := string(body)
	if !strings.Contains(payload, `"code":"UNAUTHORIZED"`) {
		t.Fatalf("expected code field, got %s", payload)
	}
	if !strings.Contains(payload, `"message":"auth required"`) {
		t.Fatalf("expected message field, got %s", payload)
	}
}

func TestResolveIdentityResponse_JSONContract(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(ResolveIdentityResponse{
		Success: true,
		Identity: IdentityContext{
			SessionID: "s1",
			UserID:    "u1",
		},
		UserInfo: UserInfo{
			Sub:   "u1",
			Group: []string{"ADMIN"},
		},
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	payload := string(body)
	if !strings.Contains(payload, `"success":true`) {
		t.Fatalf("expected success field, got %s", payload)
	}
	if !strings.Contains(payload, `"identity"`) {
		t.Fatalf("expected identity field, got %s", payload)
	}
	if !strings.Contains(payload, `"user_info"`) {
		t.Fatalf("expected user_info field, got %s", payload)
	}
}

func TestUserInfo_JSONUsesSingularGroupField(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(UserInfo{
		Sub:   "sub1",
		Group: []string{"ADMIN"},
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	payload := string(body)
	if !strings.Contains(payload, `"group":["ADMIN"]`) {
		t.Fatalf("expected singular group field, got %s", payload)
	}
	if strings.Contains(payload, `"groups"`) {
		t.Fatalf("unexpected groups field in user info payload: %s", payload)
	}
}
