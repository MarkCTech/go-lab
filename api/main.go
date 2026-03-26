package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/codemarked/go-lab/api/api"
	"github.com/codemarked/go-lab/api/auth"
	"github.com/codemarked/go-lab/api/authstore"
	"github.com/codemarked/go-lab/api/config"
	"github.com/codemarked/go-lab/api/middleware"
	"github.com/codemarked/go-lab/api/myhandlers"
	"github.com/codemarked/go-lab/api/platformrbac"
	"github.com/codemarked/go-lab/api/redisx"
	"github.com/codemarked/go-lab/api/requestid"
	"github.com/codemarked/go-lab/api/respond"
	"github.com/gin-gonic/gin"
)

func main() {
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(logHandler))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config_load_failed", "error", err)
		os.Exit(1)
	}
	gin.SetMode(cfg.GinMode)

	redisx.Init(cfg.RedisURL)
	defer redisx.Close()

	db, err := myhandlers.NewDatabase(cfg)
	if err != nil {
		slog.Error("database_init_failed", "error", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	myhandlers.AppConfig = cfg
	myhandlers.Database = db
	myhandlers.AuthStore = authstore.New(db.Db, cfg.SessionIdleTTL, cfg.SessionAbsoluteTTL)

	ts, err := auth.NewTokenService(cfg.JWTSecret, cfg.JWTSecretPrevious, cfg.JWTIssuer, cfg.JWTAudience, cfg.JWTAccessTTL)
	if err != nil {
		slog.Error("token_service_init_failed", "error", err)
		os.Exit(1)
	}
	myhandlers.TokenSvc = ts

	var oidcSvc *auth.OIDC
	if cfg.OIDCIssuerURL != "" {
		octx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		oidcSvc, err = auth.NewOIDC(octx, cfg.OIDCIssuerURL, cfg.OIDCAudience, myhandlers.AuthStore)
		cancel()
		if err != nil {
			slog.Error("oidc_init_failed", "error", err)
			os.Exit(1)
		}
		slog.Info("oidc_enabled", "issuer", cfg.OIDCIssuerURL)
	}

	if cfg.JWTActiveKeyID != "" {
		slog.Info("jwt_key_policy", "active_key_id", cfg.JWTActiveKeyID, "note", "placeholder_for_future_HS256_key_rotation")
	}

	r, err := setupRouter(cfg, oidcSvc)
	if err != nil {
		slog.Error("router_init_failed", "error", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.APIPort,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		slog.Info("server_listen", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server_failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("server_shutdown_started")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server_shutdown_error", "error", err)
	}
	slog.Info("server_stopped")
}

func setupRouter(cfg *config.Config, oidc *auth.OIDC) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery())
	r.MaxMultipartMemory = 1 << 20 // 1 MiB

	r.Use(requestid.Middleware())
	r.Use(middleware.SecurityHeaders())
	if len(cfg.CORSAllowedOrigins) > 0 {
		r.Use(middleware.DynamicCORS(cfg.CORSAllowedOrigins, cfg.CSRFHeaderName))
	}
	r.Use(legacyAPIMiddleware())

	r.GET("/healthz", myhandlers.Health)
	r.GET("/readyz", myhandlers.Ready)

	v1 := r.Group("/api/v1")
	v1.Use(middleware.CSRFCookieProtect(cfg))
	{
		registerLim := middleware.NewFixedWindowLimiter("auth_register", 15, "too many registration attempts")
		inviteAcceptLim := middleware.NewFixedWindowLimiter("auth_invite_accept", 20, "too many invite acceptance attempts")
		inviteCreateLim := middleware.NewFixedWindowLimiter("auth_invite_create", 30, "too many invite creation attempts")
		loginLim := middleware.NewFixedWindowLimiter("auth_login", 30, "too many login attempts")
		logoutLim := middleware.NewFixedWindowLimiter("auth_logout", 60, "too many logout attempts")
		refreshLim := middleware.NewFixedWindowLimiter("auth_refresh", 120, "too many session refresh attempts")
		tokenLim := middleware.NewFixedWindowLimiter("auth_token", 30, "too many token requests")
		joinTokenLim := middleware.NewFixedWindowLimiter("auth_join_token", 60, "too many join token requests")
		desktopStartLim := middleware.NewFixedWindowLimiter("auth_desktop_start", 30, "too many desktop exchange start requests")
		desktopExchangeLim := middleware.NewFixedWindowLimiter("auth_desktop_exchange", 45, "too many desktop exchange requests")
		csrfGetLim := middleware.NewFixedWindowLimiter("auth_csrf", 60, "too many csrf requests")
		changePwdLim := middleware.NewFixedWindowLimiter("auth_change_password", 10, "too many password change attempts")
		supportAckLim := middleware.NewFixedWindowLimiter("support_ack", 30, "too many support ack requests")
		backupRestoreMutLim := middleware.NewFixedWindowLimiter("backup_restore_mut", 60, "too many backup restore workflow requests")
		caseReadLim := middleware.NewFixedWindowLimiter("operator_cases_read", 120, "too many case read requests")
		caseMutLim := middleware.NewFixedWindowLimiter("operator_cases_mut", 60, "too many case mutation requests")

		v1.POST("/auth/register", registerLim, myhandlers.RegisterUser(cfg))
		v1.POST("/auth/invite/accept", inviteAcceptLim, myhandlers.AcceptOperatorInvite(cfg))
		v1.POST("/auth/login", loginLim, myhandlers.LoginUser(cfg))
		v1.POST("/auth/logout", logoutLim, myhandlers.LogoutUser(cfg))
		v1.POST("/auth/refresh", refreshLim, myhandlers.RefreshSession(cfg))
		v1.GET("/auth/csrf", csrfGetLim, myhandlers.GetAuthCSRF(cfg))

		v1.POST("/auth/change-password", changePwdLim,
			middleware.BearerOrSession(myhandlers.TokenSvc, myhandlers.AuthStore, cfg.SessionCookieName, oidc),
			myhandlers.ChangePassword(cfg),
		)
		v1.POST("/auth/join-token", joinTokenLim,
			middleware.BearerOrSession(myhandlers.TokenSvc, myhandlers.AuthStore, cfg.SessionCookieName, oidc),
			middleware.RequireHumanUser(),
			myhandlers.IssueJoinToken(cfg),
		)
		v1.POST("/auth/desktop/start", desktopStartLim,
			middleware.BearerOrSession(myhandlers.TokenSvc, myhandlers.AuthStore, cfg.SessionCookieName, oidc),
			middleware.RequireHumanUser(),
			myhandlers.StartDesktopExchangeCode(cfg),
		)

		v1.POST("/auth/token", tokenLim, myhandlers.IssueToken(cfg))
		v1.POST("/auth/bootstrap", tokenLim, myhandlers.IssueBootstrapToken(cfg))
		v1.POST("/auth/desktop/exchange", desktopExchangeLim, myhandlers.ExchangeDesktopCode(cfg))

		users := v1.Group("/users")
		users.Use(middleware.BearerOrSession(myhandlers.TokenSvc, myhandlers.AuthStore, cfg.SessionCookieName, oidc))
		users.GET("", myhandlers.GetUsers)
		users.GET("/search", myhandlers.SearchUsers)
		users.GET("/:id", myhandlers.GetUserByID)
		users.POST("", myhandlers.CreateUser)
		users.PUT("/:id", middleware.RequireHumanUser(), myhandlers.UpdateUser)
		users.DELETE("/:id",
			middleware.RequireHumanUser(),
			middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermUsersDelete),
			middleware.RequirePlatformActionReason(10),
			myhandlers.DeleteUser,
		)

		cp := v1.Group("")
		cp.Use(middleware.BearerOrSession(myhandlers.TokenSvc, myhandlers.AuthStore, cfg.SessionCookieName, oidc))
		cp.Use(middleware.RequireHumanUser())
		{
			cp.POST("/auth/invites", inviteCreateLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermSecurityWrite),
				myhandlers.CreateOperatorInvite(cfg))
			cp.GET("/players",
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermPlayersRead),
				myhandlers.ListPlayersStub)
			cp.GET("/characters",
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermCharactersRead),
				myhandlers.ListCharactersStub)
			cp.GET("/backups/status",
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermBackupsRead),
				myhandlers.GetBackupsStatus)
			cp.GET("/backups/restore-requests",
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermBackupsRead),
				myhandlers.ListBackupRestoreRequests)
			cp.POST("/backups/restore-requests", backupRestoreMutLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermBackupsRestoreRequest),
				middleware.RequirePlatformActionReason(10),
				myhandlers.PostBackupRestoreRequest)
			cp.POST("/backups/restore-requests/:id/approve", backupRestoreMutLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermBackupsRestoreApprove),
				middleware.RequirePlatformActionReason(10),
				myhandlers.PostBackupRestoreApprove)
			cp.POST("/backups/restore-requests/:id/reject", backupRestoreMutLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermBackupsRestoreApprove),
				middleware.RequirePlatformActionReason(10),
				myhandlers.PostBackupRestoreReject)
			cp.POST("/backups/restore-requests/:id/fulfill", backupRestoreMutLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermBackupsRestoreFulfill),
				middleware.RequirePlatformActionReason(10),
				myhandlers.PostBackupRestoreFulfill)
			cp.POST("/backups/restore-requests/:id/cancel", backupRestoreMutLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermBackupsRestoreRequest),
				middleware.RequirePlatformActionReason(10),
				myhandlers.PostBackupRestoreCancel)
			cp.GET("/security/me",
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermSecurityRead),
				myhandlers.GetSecurityMeRoles)
			cp.GET("/audit/admin-events",
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermAuditRead),
				myhandlers.ListAdminAuditEvents)
			cp.GET("/economy/ledger",
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermEconomyRead),
				myhandlers.GetEconomyLedger)
			cp.POST("/support/ack", supportAckLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermSupportAck),
				middleware.RequirePlatformActionReason(10),
				myhandlers.PostSupportAck)

			// Operator cases (player/character workflows; Marble applies gameplay effects out of band).
			cp.GET("/cases", caseReadLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermCasesRead),
				myhandlers.ListOperatorCases)
			cp.POST("/cases", caseMutLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermCasesWrite),
				middleware.RequirePlatformActionReason(10),
				myhandlers.PostOperatorCase)
			cp.GET("/cases/:id/notes", caseReadLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermCasesRead),
				myhandlers.ListOperatorCaseNotes)
			cp.POST("/cases/:id/notes", caseMutLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermCasesWrite),
				middleware.RequirePlatformActionReason(10),
				myhandlers.PostOperatorCaseNote)
			cp.GET("/cases/:id/actions", caseReadLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermCasesRead),
				myhandlers.ListOperatorCaseActions)
			cp.POST("/cases/:id/sanctions", caseMutLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermSanctionsWrite),
				middleware.RequirePlatformActionReason(10),
				myhandlers.PostOperatorCaseSanction)
			cp.POST("/cases/:id/recovery-requests", caseMutLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermRecoveryWrite),
				middleware.RequirePlatformActionReason(10),
				myhandlers.PostOperatorCaseRecoveryRequest)
			cp.POST("/cases/:id/appeals/resolve", caseMutLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermAppealsResolve),
				middleware.RequirePlatformActionReason(10),
				myhandlers.PostOperatorCaseAppealResolve)
			cp.GET("/cases/:id", caseReadLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermCasesRead),
				myhandlers.GetOperatorCase)
			cp.PATCH("/cases/:id", caseMutLim,
				middleware.RequirePlatformPermission(myhandlers.AuthStore, platformrbac.PermCasesWrite),
				middleware.RequirePlatformActionReason(10),
				myhandlers.PatchOperatorCase)
		}
	}

	return r, nil
}

func isUnversionedLegacyAPIPath(p string) bool {
	return p == "/api" || strings.HasPrefix(p, "/api/") && !strings.HasPrefix(p, "/api/v1")
}

// legacyAPIMiddleware returns 410 for unversioned /api/* paths (excluding /api/v1).
func legacyAPIMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isUnversionedLegacyAPIPath(c.Request.URL.Path) {
			respond.Error(c, http.StatusGone, api.CodeNotFound, "unversioned API removed; use /api/v1", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
