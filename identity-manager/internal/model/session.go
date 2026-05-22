package model

import "time"

type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	TokenExpiry  time.Time `json:"token_expiry"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type IdentityContext struct {
	SessionID  string   `json:"session_id"`
	UserID     string   `json:"user_id"`
	IdentityID string   `json:"identity_id"`
	Login      string   `json:"login"`
	Groups     []string `json:"groups"`
}

type ResolveIdentityRequest struct {
	SessionID string `json:"session_id"`
}

type ResolvedIdentity struct {
	Identity IdentityContext `json:"identity"`
	UserInfo UserInfo        `json:"user_info"`
}

type ResolveIdentityResponse struct {
	Success  bool            `json:"success"`
	Identity IdentityContext `json:"identity"`
	UserInfo UserInfo        `json:"user_info"`
}
