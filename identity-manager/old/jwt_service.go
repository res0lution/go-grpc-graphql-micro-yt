package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"portal-core/internal/logger"
	"portal-core/internal/model"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/sirupsen/logrus"
)

type JWKSManager interface {
	GetJWKS(ctx context.Context) (JWKS, error)
}

type JWKS interface {
	Len() int
	LookupKeyID(kid string) (interface{}, bool)
	Key(i int) (interface{}, bool)
}

type JWTValidator struct {
	jwksManager JWKSManager

	expectedIss string
	expectedAud string

	skipVerify bool
	logger     *logrus.Entry
}

func NewJWTValidator(jwksManager JWKSManager) *JWTValidator {
	skipVerify := os.Getenv("JWT_SKIP_SIGNATURE_VERIFICATION") == "true"

	v := &JWTValidator{
		jwksManager: jwksManager,
		expectedIss: os.Getenv("IDP_HOST"),
		expectedAud: os.Getenv("IDP_CLIENT_ID"),
		skipVerify:  skipVerify,
		logger:      logger.L().WithField("component", "jwt_validator"),
	}

	if skipVerify {
		v.logger.Warn("JWT signature verification is DISABLED")
	}

	return v
}

// -------------------- ID TOKEN --------------------

func (v *JWTValidator) ValidateIDToken(
	ctx context.Context,
	idToken string,
) (*model.IDTokenClaims, error) {

	if v.skipVerify {
		v.logger.Warn("Skipping signature verification")
		return v.parseTokenWithoutVerification(idToken)
	}

	msg, err := jws.Parse([]byte(idToken))
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWS: %w", err)
	}

	if len(msg.Signatures()) == 0 {
		return nil, errors.New("JWS has no signatures")
	}

	sig := msg.Signatures()[0]
	headers := sig.ProtectedHeaders()

	alg := headers.Algorithm()
	kid := headers.KeyID()

	v.logger.WithFields(logrus.Fields{
		"kid":       kid,
		"algorithm": alg.String(),
	}).Debug("JWS headers parsed")

	jwksSet, err := v.jwksManager.GetJWKS(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %w", err)
	}

	if jwksSet.Len() == 0 {
		return nil, errors.New("JWKS is empty")
	}

	var (
		validatedToken jwt.Token
		lastErr        error
	)

	// 1. Try by kid
	if kid != "" {
		key, found := jwksSet.LookupKeyID(kid)
		if found {
			validatedToken, err = jwt.ParseString(
				idToken,
				jwt.WithKey(alg, key),
				jwt.WithValidate(true),
			)
			if err != nil {
				lastErr = err
			}
		}
	}

	// 2. Fallback all keys
	if validatedToken == nil {
		for i := 0; i < jwksSet.Len(); i++ {
			key, ok := jwksSet.Key(i)
			if !ok {
				continue
			}

			keyAlg := jwa.RS256

			validatedToken, err = jwt.ParseString(
				idToken,
				jwt.WithKey(keyAlg, key),
				jwt.WithValidate(true),
			)

			if err == nil {
				break
			}

			lastErr = err
		}
	}

	if validatedToken == nil {
		return nil, fmt.Errorf("token validation failed: %w", lastErr)
	}

	if err := jwt.Validate(validatedToken, jwt.WithClock(jwt.ClockFunc(time.Now))); err != nil {
		return nil, fmt.Errorf("token time validation failed: %w", err)
	}

	return v.extractIDTokenClaims(validatedToken)
}

// -------------------- ACCESS TOKEN --------------------

func (v *JWTValidator) ValidateAccessToken(
	ctx context.Context,
	accessToken string,
) error {

	if v.skipVerify {
		v.logger.Warn("Skipping access token verification")
		return nil
	}

	msg, err := jws.Parse([]byte(accessToken))
	if err != nil {
		return fmt.Errorf("failed to parse JWS: %w", err)
	}

	if len(msg.Signatures()) == 0 {
		return errors.New("no signatures in access token")
	}

	sig := msg.Signatures()[0]
	headers := sig.ProtectedHeaders()

	kid := headers.KeyID()
	alg := headers.Algorithm()

	jwksSet, err := v.jwksManager.GetJWKS(ctx)
	if err != nil {
		return fmt.Errorf("failed to get JWKS: %w", err)
	}

	var lastErr error

	if kid != "" {
		if key, found := jwksSet.LookupKeyID(kid); found {
			if _, err := jwt.ParseString(
				accessToken,
				jwt.WithKey(alg, key),
				jwt.WithValidate(true),
			); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
	}

	for i := 0; i < jwksSet.Len(); i++ {
		key, ok := jwksSet.Key(i)
		if !ok {
			continue
		}
		if _, err := jwt.ParseString(
			accessToken,
			jwt.WithKey(jwa.RS256, key),
			jwt.WithValidate(true),
		); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}

	return fmt.Errorf("access token validation failed: %w", lastErr)
}

// -------------------- LOGOUT TOKEN --------------------

func (v *JWTValidator) ValidateLogoutToken(
	ctx context.Context,
	logoutToken string,
) (*model.LogoutTokenClaims, error) {

	if v.skipVerify {
		return v.parseLogoutTokenWithoutVerification(logoutToken)
	}

	msg, err := jws.Parse([]byte(logoutToken))
	if err != nil {
		return nil, fmt.Errorf("failed to parse logout token: %w", err)
	}

	if len(msg.Signatures()) == 0 {
		return nil, errors.New("logout token has no signatures")
	}

	sig := msg.Signatures()[0]
	headers := sig.ProtectedHeaders()

	kid := headers.KeyID()
	alg := headers.Algorithm()

	jwksSet, err := v.jwksManager.GetJWKS(ctx)
	if err != nil {
		return nil, err
	}

	var (
		validatedToken jwt.Token
		lastErr        error
	)

	if kid != "" {
		if key, found := jwksSet.LookupKeyID(kid); found {
			validatedToken, err = jwt.ParseString(
				logoutToken,
				jwt.WithKey(alg, key),
				jwt.WithValidate(true),
			)
			if err != nil {
				lastErr = err
			}
		}
	}

	if validatedToken == nil {
		for i := 0; i < jwksSet.Len(); i++ {
			key, ok := jwksSet.Key(i)
			if !ok {
				continue
			}

			if validatedToken, err = jwt.ParseString(
				logoutToken,
				jwt.WithKey(jwa.RS256, key),
				jwt.WithValidate(true),
			); err == nil {
				break
			}

			lastErr = err
		}
	}

	if validatedToken == nil {
		return nil, fmt.Errorf("logout token validation failed: %w", lastErr)
	}

	return v.extractLogoutClaims(validatedToken)
}

// -------------------- PARSE HELPERS --------------------

func (v *JWTValidator) parseTokenWithoutVerification(
	token string,
) (*model.IDTokenClaims, error) {

	t, err := jwt.ParseString(
		token,
		jwt.WithVerify(false),
		jwt.WithValidate(false),
	)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	return v.extractIDTokenClaims(t)
}

func (v *JWTValidator) parseLogoutTokenWithoutVerification(
	token string,
) (*model.LogoutTokenClaims, error) {

	t, err := jwt.ParseString(
		token,
		jwt.WithVerify(false),
		jwt.WithValidate(false),
	)
	if err != nil {
		return nil, err
	}

	return v.extractLogoutClaims(t)
}

// -------------------- CLAIMS --------------------

func (v *JWTValidator) extractIDTokenClaims(t jwt.Token) (*model.IDTokenClaims, error) {
	c := &model.IDTokenClaims{}

	c.Sub = t.Subject()
	c.Iss = t.Issuer()
	c.Aud = t.Audience()

	if exp := t.Expiration(); !exp.IsZero() {
		c.Exp = exp.Unix()
	}

	if iat := t.IssuedAt(); !iat.IsZero() {
		c.Iat = iat.Unix()
	}

	if c.Sub == "" {
		return nil, errors.New("missing sub claim")
	}

	return c, nil
}

func (v *JWTValidator) extractLogoutClaims(t jwt.Token) (*model.LogoutTokenClaims, error) {
	c := &model.LogoutTokenClaims{}

	c.Sub = t.Subject()
	c.Iss = t.Issuer()
	c.Jti = t.JwtID()
	c.Aud = t.Audience()

	if sid, ok := t.Get("sid"); ok {
		if s, ok := sid.(string); ok {
			c.Sid = s
		}
	}

	return c, nil
}
