package model

import "time"

type AuthRequest struct {
	Scope string `form:"scope"`
	State string `form:"state"`
	Nonce string `form:"nonce"`
}

type AuthCallbackQuery struct {
	Code             string `form:"code"`
	State            string `form:"state"`
	Error            string `form:"error"`
	ErrorDescription string `form:"error_description"`
}

type OAuth2TokenExchange struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

type IDTokenClaims struct {
	Iss            string   `json:"iss"`
	Sub            string   `json:"sub"`
	Exp            int64    `json:"exp"`
	Iat            int64    `json:"iat"`
	Aud            []string `json:"aud"`
	ACR            string   `json:"acr"`
	AMR            []string `json:"amr"`
	IdentityID     string   `json:"identity_id"`
	Login          string   `json:"login"`
	Email          string   `json:"email"`
	Nonce          string   `json:"nonce"`
	GivenName      string   `json:"given_name"`
	FamilyName     string   `json:"family_name"`
	Name           string   `json:"name"`
	Group          []string `json:"group"`
	WinAccountName string   `json:"winaccountname"`
	EmployeeNumber string   `json:"employee_number"`
}

type LogoutTokenClaims struct {
	Sub    string                 `json:"sub"`
	Sid    string                 `json:"sid"`
	Events map[string]any         `json:"events"`
}

type UserInfo struct {
	Sub            string   `json:"sub"`
	Iss            string   `json:"iss,omitempty"`
	Aud            []string `json:"aud,omitempty"`
	ACR            string   `json:"acr,omitempty"`
	AMR            []string `json:"amr,omitempty"`
	Iat            int64    `json:"iat,omitempty"`
	IdentityID     string   `json:"identity_id"`
	Login          string   `json:"login"`
	Email          string   `json:"email"`
	GivenName      string   `json:"given_name"`
	FamilyName     string   `json:"family_name"`
	Name           string   `json:"name"`
	// IDP token/userinfo payload uses singular "group" claim.
	Group          []string `json:"group"`
	WinAccountName string   `json:"winaccountname"`
	EmployeeNumber string   `json:"employee_number"`
}

func (u UserInfo) HasGroup(groupName string) bool {
	for _, group := range u.Group {
		if group == groupName {
			return true
		}
	}
	return false
}

func (u UserInfo) HasAnyGroup(groups []string) bool {
	for _, requiredGroup := range groups {
		if u.HasGroup(requiredGroup) {
			return true
		}
	}
	return false
}

func (u UserInfo) HasAllGroups(groups []string) bool {
	for _, requiredGroup := range groups {
		if !u.HasGroup(requiredGroup) {
			return false
		}
	}
	return true
}

type LoginResult struct {
	SessionID string    `json:"session_id"`
	ExpiresAt time.Time `json:"expires_at"`
	Redirect  string    `json:"redirect"`
}
