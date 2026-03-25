package myhandlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/codemarked/go-lab/api/api"
	"github.com/codemarked/go-lab/api/auth"
	"github.com/codemarked/go-lab/api/config"
	"github.com/codemarked/go-lab/api/respond"
	"github.com/gin-gonic/gin"
)

// ChangePassword updates the password for the authenticated end-user (cookie or Bearer with subject user:N).
// Revokes all sessions and clears cookies so the client must log in again.
func ChangePassword(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthStore == nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "auth store not configured", nil)
			return
		}
		sub := strings.TrimSpace(c.GetString("auth_subject"))
		if !strings.HasPrefix(sub, "user:") {
			respond.Error(c, http.StatusForbidden, api.CodeForbidden, "password change requires a user session", nil)
			return
		}
		uid, err := strconv.Atoi(strings.TrimPrefix(sub, "user:"))
		if err != nil || uid <= 0 {
			respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid session subject", nil)
			return
		}
		var body struct {
			CurrentPassword string `json:"current_password" binding:"required"`
			NewPassword     string `json:"new_password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid request body", map[string]any{"field": "body"})
			return
		}
		if err := auth.ValidatePasswordLength(body.NewPassword); err != nil {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "new password does not meet policy", map[string]any{"field": "new_password", "min_len": 8})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()
		hash, err := AuthStore.GetUserPasswordHashByID(ctx, uid)
		if errors.Is(err, sql.ErrNoRows) {
			respond.Error(c, http.StatusNotFound, api.CodeNotFound, "user not found", nil)
			return
		}
		if err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to load credentials", nil)
			return
		}
		if strings.TrimSpace(hash) == "" {
			respond.Error(c, http.StatusForbidden, api.CodeForbidden, "password login not enabled for this account", nil)
			return
		}
		ok, err := auth.VerifyPassword(body.CurrentPassword, hash)
		if err != nil || !ok {
			respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "current password incorrect", nil)
			return
		}
		newHash, err := auth.HashPassword(body.NewPassword)
		if err != nil {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "new password does not meet policy", nil)
			return
		}
		if err := AuthStore.UpdateUserPasswordHash(ctx, uid, newHash); err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to update password", nil)
			return
		}
		if _, err := AuthStore.RevokeAllSessionsForUser(ctx, uid); err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to revoke sessions", nil)
			return
		}
		_ = AuthStore.InsertAudit(ctx, "auth_password_changed", &uid, clientIP(c), c.Request.UserAgent(), "", nil)
		clearSessionCookie(c, cfg)
		clearCSRFCookie(c, cfg)
		respond.JSONOK(c, http.StatusOK, gin.H{"password_changed": true, "relogin_required": true})
	}
}
