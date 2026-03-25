package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenService issues and verifies access tokens (HS256).
type TokenService struct {
	secret     []byte
	prevSecret []byte // optional; ParseAccessToken accepts signatures from either during rotation
	issuer     string
	audience   string
	ttl        time.Duration
}

// Claims embedded in access JWTs.
type Claims struct {
	jwt.RegisteredClaims
	TokenUse      string `json:"token_use,omitempty"`
	JoinSessionID string `json:"join_session_id,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
}

// NewTokenService builds a verifier/issuer. secret must be non-empty.
// previousSecret may be empty; when set it must be at least 32 bytes and tokens verify against either secret until rotation completes.
func NewTokenService(secret, previousSecret, issuer, audience string, ttl time.Duration) (*TokenService, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT secret too short")
	}
	if issuer == "" || audience == "" {
		return nil, errors.New("issuer and audience required")
	}
	var prev []byte
	if strings.TrimSpace(previousSecret) != "" {
		if len(previousSecret) < 32 {
			return nil, errors.New("previous JWT secret too short")
		}
		prev = []byte(previousSecret)
	}
	return &TokenService{
		secret:     []byte(secret),
		prevSecret: prev,
		issuer:     issuer,
		audience:   audience,
		ttl:        ttl,
	}, nil
}

// MintAccessToken returns a signed JWT access token.
func (s *TokenService) MintAccessToken(subject string) (string, time.Time, error) {
	if subject == "" {
		return "", time.Time{}, errors.New("subject required")
	}
	now := time.Now().UTC()
	exp := now.Add(s.ttl)
	jti, err := randomJTI()
	if err != nil {
		return "", time.Time{}, err
	}
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   subject,
			Audience:  jwt.ClaimStrings{s.audience},
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        jti,
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	signed, err := t.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// MintJoinToken returns a short-lived JWT for Marble join handoff.
func (s *TokenService) MintJoinToken(subject, joinSessionID string, ttl time.Duration) (string, time.Time, error) {
	if subject == "" {
		return "", time.Time{}, errors.New("subject required")
	}
	joinSessionID = strings.TrimSpace(joinSessionID)
	if joinSessionID == "" {
		return "", time.Time{}, errors.New("join session id required")
	}
	if ttl <= 0 {
		return "", time.Time{}, errors.New("join token ttl must be positive")
	}
	now := time.Now().UTC()
	exp := now.Add(ttl)
	jti, err := randomJTI()
	if err != nil {
		return "", time.Time{}, err
	}
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   subject,
			Audience:  jwt.ClaimStrings{s.audience},
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        jti,
		},
		TokenUse:      "join",
		JoinSessionID: joinSessionID,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	signed, err := t.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// MintDesktopAccessToken returns a short-lived JWT for desktop user API calls.
func (s *TokenService) MintDesktopAccessToken(subject, sessionID string) (string, time.Time, error) {
	if subject == "" {
		return "", time.Time{}, errors.New("subject required")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", time.Time{}, errors.New("session id required")
	}
	now := time.Now().UTC()
	exp := now.Add(s.ttl)
	jti, err := randomJTI()
	if err != nil {
		return "", time.Time{}, err
	}
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   subject,
			Audience:  jwt.ClaimStrings{s.audience},
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        jti,
		},
		TokenUse:  "desktop_access",
		SessionID: sessionID,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	signed, err := t.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// ParseAccessToken validates signature, expiry, issuer, and audience.
func (s *TokenService) ParseAccessToken(tokenString string) (*Claims, error) {
	claims, err := s.parseAccessTokenWithSecret(tokenString, s.secret)
	if err == nil {
		return claims, nil
	}
	if len(s.prevSecret) > 0 {
		claims, prevErr := s.parseAccessTokenWithSecret(tokenString, s.prevSecret)
		if prevErr == nil {
			return claims, nil
		}
	}
	return nil, err
}

func (s *TokenService) parseAccessTokenWithSecret(tokenString string, secret []byte) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return secret, nil
	}, jwt.WithIssuer(s.issuer), jwt.WithAudience(s.audience), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

func randomJTI() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
