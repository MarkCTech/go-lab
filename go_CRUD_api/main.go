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

	"github.com/codemarked/go-lab/go_CRUD_api/api"
	"github.com/codemarked/go-lab/go_CRUD_api/auth"
	"github.com/codemarked/go-lab/go_CRUD_api/config"
	"github.com/codemarked/go-lab/go_CRUD_api/middleware"
	"github.com/codemarked/go-lab/go_CRUD_api/myhandlers"
	"github.com/codemarked/go-lab/go_CRUD_api/requestid"
	"github.com/codemarked/go-lab/go_CRUD_api/respond"
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

	db, err := myhandlers.NewDatabase(cfg)
	if err != nil {
		slog.Error("database_init_failed", "error", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	myhandlers.AppConfig = cfg
	myhandlers.Database = db

	ts, err := auth.NewTokenService(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.JWTAccessTTL)
	if err != nil {
		slog.Error("token_service_init_failed", "error", err)
		os.Exit(1)
	}
	myhandlers.TokenSvc = ts

	r := setupRouter(cfg)

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

func setupRouter(cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.MaxMultipartMemory = 1 << 20 // 1 MiB

	r.Use(requestid.Middleware())
	r.Use(middleware.SecurityHeaders())
	if len(cfg.CORSAllowedOrigins) > 0 {
		r.Use(middleware.DynamicCORS(cfg.CORSAllowedOrigins))
	}
	r.Use(legacyAPIMiddleware())

	r.GET("/healthz", myhandlers.Health)
	r.GET("/readyz", myhandlers.Ready)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/token", middleware.NewTokenEndpointLimiter(30), myhandlers.IssueToken(cfg))
		v1.POST("/auth/bootstrap", middleware.NewTokenEndpointLimiter(30), myhandlers.IssueBootstrapToken(cfg))

		users := v1.Group("/users")
		users.Use(middleware.BearerAuth(myhandlers.TokenSvc))
		users.GET("", myhandlers.GetUsers)
		users.GET("/search", myhandlers.SearchUsers)
		users.GET("/:id", myhandlers.GetUserByID)
		users.POST("", myhandlers.CreateUser)
		users.PUT("/:id", myhandlers.UpdateUser)
		users.DELETE("/:id", myhandlers.DeleteUser)
	}

	return r
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
