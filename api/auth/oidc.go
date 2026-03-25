// Package auth: OIDC access-token verification (e.g. Auth0) via github.com/coreos/go-oidc/v3.
// Rationale: maintained OIDC discovery + JWKS rotation; avoids hand-rolling RS256/JWKS fetch.
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"github.com/codemarked/go-lab/api/authstore"
	"github.com/coreos/go-oidc/v3/oidc"
)

// OIDC verifies RS256 access JWTs from an external issuer and maps humans to local user ids.
type OIDC struct {
	verifier   *oidc.IDTokenVerifier
	issuerURL  string
	store     *authstore.Store
	audience  string
}

// NewOIDC builds a verifier from issuer discovery. issuerURL must match token `iss` (e.g. https://tenant.auth0.com/).
func NewOIDC(ctx context.Context, issuerURL, audience string, store *authstore.Store) (*OIDC, error) {
	issuerURL = strings.TrimSpace(issuerURL)
	audience = strings.TrimSpace(audience)
	if issuerURL == "" || audience == "" {
		return nil, fmt.Errorf("oidc: issuer and audience required")
	}
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("oidc provider: %w", err)
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: audience})
	return &OIDC{
		verifier:  verifier,
		issuerURL: issuerURL,
		audience:  audience,
		store:     store,
	}, nil
}

// TryResolveAuthSubject verifies the raw JWT and returns an auth_subject (user:N or client:id).
// Errors mean the token is not a valid OIDC access token for this audience.
func (o *OIDC) TryResolveAuthSubject(ctx context.Context, rawToken string) (authSubject string, err error) {
	if o == nil || o.verifier == nil {
		return "", fmt.Errorf("oidc not configured")
	}
	idt, err := o.verifier.Verify(ctx, rawToken)
	if err != nil {
		return "", err
	}
	sub := strings.TrimSpace(idt.Subject)
	if sub == "" {
		return "", fmt.Errorf("empty sub")
	}

	// Auth0 client-credentials: sub is "{clientId}@clients"
	if strings.HasSuffix(sub, "@clients") {
		cid := strings.TrimSuffix(sub, "@clients")
		if cid == "" {
			return "", fmt.Errorf("empty client id in m2m sub")
		}
		return "client:" + cid, nil
	}

	if o.store == nil {
		return "", fmt.Errorf("auth store required for user OIDC mapping")
	}

	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	_ = idt.Claims(&claims)
	display := strings.TrimSpace(claims.Name)
	if display == "" {
		display = strings.TrimSpace(claims.Email)
	}
	iss := strings.TrimSpace(idt.Issuer)
	if iss == "" {
		iss = o.issuerURL
	}

	uid, err := o.store.EnsureOIDCUser(ctx, iss, sub, display)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("user:%d", uid), nil
}

// LogReject logs a failed OIDC verification without leaking the raw token.
func LogReject(requestID string, rawToken string, err error) {
	h := sha256.Sum256([]byte(strings.TrimSpace(rawToken)))
	slog.Warn("oidc_bearer_rejected",
		"request_id", requestID,
		"token_sha256_prefix", hex.EncodeToString(h[:8]),
		"reason", err.Error(),
	)
}
