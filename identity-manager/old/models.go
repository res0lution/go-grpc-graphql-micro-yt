package model

import "time"

// OAuth authorization request
type AuthRequest struct {
 Scope        string form:"scope"          // requested scopes (openid offline)
 ResponseType string form:"response_type"  // code
 ClientID     string form:"client_id"
 RedirectURI  string form:"redirect_uri"
 State        string form:"state"          // random string (min 8 chars)
 Nonce        string form:"nonce"          // random string for ID token
}

// OAuth authorization response
type AuthResponse struct {
 Success bool   json:"success"
 AuthURL string json:"auth_url,omitempty" // URL for IDP redirect
 Error   string json:"error,omitempty"
}

// User model
type User struct {
 ID              string    json:"id"
 IdentityID      string    json:"identity_id"
 Sub             string    json:"sub"
 Login           string    json:"login"
 Email           string    json:"email"
 EmployeeNumber  string    json:"employee_number"
 WinAccountName  string    json:"winaccountname"
 GivenName       string    json:"given_name"
 FamilyName      string    json:"family_name"
 Name            string    json:"name"
 Groups          []string  json:"groups" // AD groups
 IsActive        bool      json:"is_active"
 LastLoginAt     time.Time json:"last_login_at"
 CreatedAt       time.Time json:"created_at"
 UpdatedAt       time.Time json:"updated_at"
}

// User create request
type UserCreateRequest struct {
 IdentityID     string   json:"identity_id" binding:"required"
 Sub            string   json:"sub" binding:"required"
 Login          string   json:"login"
 Email          string   json:"email"
 EmployeeNumber string   json:"employee_number"
 WinAccountName string   json:"winaccountname"
 GivenName      string   json:"given_name"
 FamilyName     string   json:"family_name"
 Name           string   json:"name"
 Groups         []string json:"groups"
}

// Session model
type Session struct {
 ID           string    json:"id"
 UserID       string    json:"user_id"
 AccessToken  string    json:"access_token"
 RefreshToken string    json:"refresh_token"
 IDToken      string    json:"id_token"
 ExpiresAt    time.Time json:"expires_at"
 TokenExpiry  time.Time json:"token_expiry"
 CreatedAt    time.Time json:"created_at"
 UpdatedAt    time.Time json:"updated_at"
}

// OAuth2 token exchange
type OAuth2TokenExchange struct {
 AccessToken  string json:"access_token"
 ExpiresIn    int    json:"expires_in"
 IDToken      string json:"id_token"
 RefreshToken string json:"refresh_token"
 TokenType    string json:"token_type"
 Scope        string json:"scope"
}

// ID token claims
type IDTokenClaims struct {
 // Basic claims
 Iss string   json:"iss" // Issuer
 Sub string   json:"sub" // Subject
 Exp int64    json:"exp" // Expiration time
 Iat int64    json:"iat" // Issued at
 Aud []string json:"aud" // Audience

 // Authentication info
 ACR string   json:"acr" // Authentication Context Reference
 AMR []string json:"amr" // Authentication Methods References

 // User info
 IdentityID     string   json:"identity_id"
 Login          string   json:"login"
 GivenName      string   json:"given_name"
 FamilyName     string   json:"family_name"
 Email          string   json:"email"
 Group          []string json:"group"
 Name           string   json:"name"
 WinAccountName string   json:"winaccountname"
 EmployeeNumber string   json:"employee_number"

 // Optional USS User claims
 SelectedUssUserType      string json:"selected_uss_user_type,omitempty"
 SelectedUserCtn          string json:"selected_user_ctn,omitempty"
 SelectedUserLogin        string json:"selected_user_login,omitempty"
 SelectedFttbConvergentLog string json:"selected_fttb_convergent_login,omitempty"
 MainUserCtn              string json:"main_user_ctn,omitempty"
 MainUserLogin            string json:"main_user_login,omitempty"
 MainFttbConvergentLogin  string json:"main_fttb_convergent_login,omitempty"

 ParentSelectedLogin string json:"parent_selected_login,omitempty"

 // Optional CTN scope
 UserCtn string json:"user_ctn,omitempty"

 // Support user scope
 SsoImpersonateUser string json:"sso_impersonate_user,omitempty"
}

// User info
type UserInfo struct {
 EmployeeNumber string   json:"employee_number"
 ACR            string   json:"acr"
 AMR            []string json:"amr"
 Aud            []string json:"aud"
 AuthTime       int64    json:"auth_time"
 Email          string   json:"email"
 FamilyName     string   json:"family_name"
 GivenName      string   json:"given_name"
 Group          []string json:"group"
 Iat            int64    json:"iat"
 IdentityID     string   json:"identity_id"
 Iss            string   json:"iss"
 Login          string   json:"login"
 Rat            int64    json:"rat"
 Sub            string   json:"sub"
 WinAccountName string   json:"winaccountname"
 Name           string   json:"name,omitempty"

 SelectedUssUserType       string json:"selected_uss_user_type,omitempty"
 SelectedUserCtn           string json:"selected_user_ctn,omitempty"
 SelectedUserLogin         string json:"selected_user_login,omitempty"
 SelectedFttbConvergentLog string json:"selected_fttb_convergent_login,omitempty"

 MainUssUserType         string json:"main_uss_user_type,omitempty"
 MainUserCtn             string json:"main_user_ctn,omitempty"
 MainUserLogin           string json:"main_user_login,omitempty"
 MainFttbConvergentLogin string json:"main_fttb_convergent_login,omitempty"

 ParentSelectedLogin string json:"parent_selected_login,omitempty"
 UserCtn             string json:"user_ctn,omitempty"
 SsoImpersonateUser  string json:"sso_impersonate_user,omitempty"
}

// API error
type APIError struct {
 Code    string json:"code"
 Message string json:"message"
}

// User info response
type UserInfoResponse struct {
 Success bool       json:"success"
 Message string     json:"message"
 User    UserInfo   json:"user,omitempty"
 Error   *APIError  json:"error,omitempty"
}

// Logout token claims
type LogoutTokenClaims struct {
 Iss    string                 json:"iss"
 Sub    string                 json:"sub"
 Aud    []string               json:"aud"
 Iat    int64                  json:"iat"
 Jti    string                 json:"jti"
 Sid    string                 json:"sid"
 Events map[string]interface{} json:"events"
}

// Helpers

func (c IDTokenClaims) IsSystemAccount() bool {
 return c.ACR == "winaccount_service_login" ||
  c.Login == "SYS_LOGIN" ||
  c.WinAccountName == "SYS_LOGIN"
}

func (c IDTokenClaims) IsADFSUser() bool {
 return c.WinAccountName != "" ||
  len(c.Group) > 0 ||
  c.Email != ""
}

func (u UserInfo) GetFullName() string {
 if u.GivenName != "" && u.FamilyName != "" {
  return u.GivenName + " " + u.FamilyName
 }

 if u.GivenName != "" {
  return u.GivenName
 }

 if u.FamilyName != "" {
  return u.FamilyName
 }

 return u.Login
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

func (l LogoutTokenClaims) IsBackChannelLogout() bool {
 if l.Events == nil {
  return false
 }

 _, ok := l.Events["http://schemas.openid.net/event/backchannel-logout"]
 return ok
}