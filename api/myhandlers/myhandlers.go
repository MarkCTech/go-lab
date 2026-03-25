package myhandlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/codemarked/go-lab/api/config"
	"github.com/codemarked/go-lab/api/requestid"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type DBConn struct {
	Db *sql.DB
}

type User struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Pennies int    `json:"pennies"`
}

var Database *DBConn

// AppConfig is set from main for readiness checks (migration version).
var AppConfig *config.Config

// NewDatabase opens a connection to the configured database.
// Schema must exist (e.g. MySQL init or migrations); the API never creates or alters schema.
func NewDatabase(cfg *config.Config) (*DBConn, error) {
	if cfg == nil {
		return nil, errors.New("nil config")
	}
	db, err := sql.Open("mysql", dsnFromConfig(cfg))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database ping failed (ensure DB exists and migrations ran): %w", err)
	}

	return &DBConn{Db: db}, nil
}

func (d *DBConn) Close() error {
	if d == nil || d.Db == nil {
		return nil
	}
	return d.Db.Close()
}

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func Ready(c *gin.Context) {
	rid := c.GetHeader(requestid.Header)
	if Database == nil || Database.Db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": "no_database", "request_id": rid})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := Database.Db.PingContext(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": "db_ping_failed", "error": err.Error(), "request_id": rid})
		return
	}

	if AppConfig != nil && AppConfig.MigrationExpectedVersion > 0 {
		var version int
		var dirty bool
		err := Database.Db.QueryRowContext(ctx,
			"SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&version, &dirty)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": "migrations_not_applied", "request_id": rid})
				return
			}
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": "migration_check_failed", "error": err.Error(), "request_id": rid})
			return
		}
		if dirty {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": "migration_dirty", "version": version, "request_id": rid})
			return
		}
		if version < AppConfig.MigrationExpectedVersion {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready", "reason": "migration_version_mismatch",
				"version": version, "expected_min": AppConfig.MigrationExpectedVersion, "request_id": rid,
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready", "request_id": rid})
}

func parseID(idParam string) (int, error) {
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		return 0, errors.New("id must be a positive integer")
	}
	return id, nil
}

func dsnFromConfig(cfg *config.Config) string {
	// parseTime + UTC: without parseTime, TIMESTAMP/DATETIME scans break session expiry checks (401 after login).
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=UTC&charset=utf8mb4",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)
}
