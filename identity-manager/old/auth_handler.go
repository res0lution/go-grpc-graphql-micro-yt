package handler
	
import (
 "encoding/json"
 "io"
 "net/http"
 "net/url"
 "os"
 "strings"
 "time"

 "portal-core/internal/logger"
 "portal-core/internal/model"
 "portal-core/internal/service"

 "github.com/gin-gonic/gin"
 "github.com/sirupsen/logrus"
)

type AuthHandler struct {
 AuthService service.AuthService
 logger      *logrus.Entry
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
 return &AuthHandler{
  AuthService: authService,
  logger:      logger.L().WithField("component", "auth_handler"),
 }
}

// -------------------- INITIATE AUTH --------------------

func (h *AuthHandler) InitiateAuth(ctx *gin.Context) {
 var req model.AuthRequest

 // Generate state if missing
 if req.State == "" {
  state, err := h.AuthService.GenerateState()
  if err != nil {
   h.logger.WithError(err).Error("failed to generate state")
   ctx.JSON(http.StatusInternalServerError, model.AuthResponse{
    Success:      false,
    Error:        "server_error",
    ErrorMessage: "Failed to generate state",
   })
   return
  }
  req.State = state
 }

 // Generate nonce if missing
 if req.Nonce == "" {
  nonce, err := h.AuthService.GenerateNonce()
  if err != nil {
   h.logger.WithError(err).Error("failed to generate nonce")
   ctx.JSON(http.StatusInternalServerError, model.AuthResponse{
    Success:      false,
    Error:        "server_error",
    ErrorMessage: "Failed to generate nonce",
   })
   return
  }
  req.Nonce = nonce
 }

 h.logger.WithFields(logrus.Fields{
  "scope": req.Scope,
  "state": req.State,
 }).Debug("initiating oauth flow")

 if err := h.AuthService.ValidateAuthRequest(&req); err != nil {
  h.logger.WithError(err).Error("invalid auth request")

  ctx.JSON(http.StatusBadRequest, model.AuthResponse{
   Success:      false,
   Error:        "invalid_request",
   ErrorMessage: err.Error(),
  })
  return
 }

 authURL, err := h.AuthService.BuildAuthURL(&req)
 if err != nil {
  h.logger.WithError(err).Error("failed to build auth url")

  ctx.JSON(http.StatusInternalServerError, model.AuthResponse{
   Success:      false,
   Error:        "server_error",
   ErrorMessage: "Failed to build authorization URL",
  })
  return
 }

 ctx.Redirect(http.StatusFound, authURL)
}

// -------------------- CALLBACK --------------------

func (h *AuthHandler) HandleCallback(ctx *gin.Context) {
 type CallbackQuery struct {
  Code             string form:"code"
  State            string form:"state"
  Error            string form:"error"
  ErrorDescription string form:"error_description"
 }

 var q CallbackQuery

 if err := ctx.ShouldBindQuery(&q); err != nil {
  ctx.JSON(http.StatusBadRequest, gin.H{
   "status":  "error",
   "message": "invalid callback format",
  })
  return
 }

 if q.Error != "" {
  ctx.JSON(http.StatusUnauthorized, gin.H{
   "status":             "error",
   "error":              q.Error,
   "error_description":  q.ErrorDescription,
  })
  return
 }

 if q.Code == "" || q.State == "" {
  ctx.JSON(http.StatusBadRequest, gin.H{
   "status":  "error",
   "message": "missing code or state",
  })
  return
 }

 // env validation
 clientID := os.Getenv("IDP_CLIENT_ID")
 clientSecret := os.Getenv("IDP_CLIENT_SECRET")
 idpHost := os.Getenv("IDP_HOST")
 redirectURI := os.Getenv("PORTAL_UI_REDIRECT_URI")

 if clientID == ""  clientSecret == ""  idpHost == "" || redirectURI == "" {
  h.logger.Error("missing idp env vars")
  ctx.Status(http.StatusInternalServerError)
  return
 }

 form := url.Values{}
 form.Set("grant_type", "authorization_code")
 form.Set("code", q.Code)
 form.Set("client_id", clientID)
 form.Set("client_secret", clientSecret)
 form.Set("redirect_uri", redirectURI)

 tokenURL := idpHost + "/oauth2/token"

 req, err := http.NewRequest("POST", tokenURL, strings.NewReader(form.Encode()))
 if err != nil {
  h.logger.WithError(err).Error("failed to create request")
  ctx.Status(http.StatusInternalServerError)
  return
 }

 req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
 req.Header.Set("Accept", "application/json")
[19.05.2026 20:52] Yaroslav Plotnickov: client := &http.Client{}
 resp, err := client.Do(req)
 if err != nil {
  h.logger.WithError(err).Error("token request failed")
  ctx.Status(http.StatusInternalServerError)
  return
 }
 defer resp.Body.Close()

 body, err := io.ReadAll(resp.Body)
 if err != nil {
  ctx.Status(http.StatusInternalServerError)
  return
 }

 if resp.StatusCode != http.StatusOK {
  h.logger.WithField("status", resp.StatusCode).Error("idp returned error")
  ctx.String(resp.StatusCode, "idp error")
  return
 }

 var token model.OAuth2TokenExchange
 if err := json.Unmarshal(body, &token); err != nil {
  h.logger.WithError(err).Error("invalid token response")
  ctx.Status(http.StatusInternalServerError)
  return
 }

 sessionID, expiresAt, err := h.AuthService.CreateSession(ctx.Request.Context(), &token)
 if err != nil {
  h.logger.WithError(err).Error("failed to create session")
  ctx.Status(http.StatusInternalServerError)
  return
 }

 http.SetCookie(ctx.Writer, &http.Cookie{
  Name:     "session_id",
  Value:    sessionID,
  Path:     "/",
  Expires:  expiresAt,
  HttpOnly: true,
  Secure:   true,
  SameSite: http.SameSiteLaxMode,
  MaxAge:   int(time.Until(expiresAt).Seconds()),
 })

 ctx.Redirect(http.StatusFound, os.Getenv("PORTAL_UI_HOST"))
}

// -------------------- LOGOUT --------------------

func (h *AuthHandler) HandleLogout(ctx *gin.Context) {
 if err := ctx.Request.ParseForm(); err != nil {
  ctx.JSON(http.StatusBadRequest, gin.H{
   "error": "invalid_request",
  })
  return
 }

 logoutToken := ctx.Request.Form.Get("logout_token")
 if logoutToken == "" {
  ctx.JSON(http.StatusBadRequest, gin.H{
   "error": "missing_logout_token",
  })
  return
 }

 claims, err := h.AuthService.ValidateLogoutToken(ctx.Request.Context(), logoutToken)
 if err != nil {
  ctx.JSON(http.StatusUnauthorized, gin.H{
   "error": "invalid_token",
  })
  return
 }

 if err := h.AuthService.ProcessBackChannelLogout(ctx.Request.Context(), claims); err != nil {
  ctx.JSON(http.StatusInternalServerError, gin.H{
   "error": "server_error",
  })
  return
 }

 ctx.Status(http.StatusOK)
}

// -------------------- JWKS --------------------

func (h *AuthHandler) GetJWKSStatus(ctx *gin.Context) {
 stats := h.AuthService.GetJWKSStats()

 ctx.JSON(http.StatusOK, gin.H{
  "success": true,
  "jwks":    stats,
 })
}