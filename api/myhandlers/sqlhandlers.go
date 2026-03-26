package myhandlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/codemarked/go-lab/api/api"
	"github.com/codemarked/go-lab/api/middleware"
	"github.com/codemarked/go-lab/api/requestid"
	"github.com/codemarked/go-lab/api/respond"
	"github.com/gin-gonic/gin"
)

type userPayload struct {
	Name    string `json:"name" binding:"required,min=1,max=100"`
	Pennies int    `json:"pennies" binding:"gte=0"`
}

func GetUsers(c *gin.Context) {
	rows, err := Database.Db.Query(
		"SELECT id, name, pennies FROM users WHERE deleted_at IS NULL ORDER BY pennies DESC, id ASC",
	)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to query users", nil)
		return
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Pennies); err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to decode users", nil)
			return
		}
		users = append(users, u)
	}

	respond.OK(c, users)
}

func SearchUsers(c *gin.Context) {
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		respond.OK(c, []User{})
		return
	}
	rows, err := Database.Db.Query(
		"SELECT id, name, pennies FROM users WHERE deleted_at IS NULL AND name LIKE CONCAT('%', ?, '%') ORDER BY pennies DESC, id ASC",
		name,
	)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to search users", nil)
		return
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Pennies); err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to decode users", nil)
			return
		}
		users = append(users, u)
	}
	respond.OK(c, users)
}

func GetUserByID(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, err.Error(), map[string]any{"field": "id"})
		return
	}

	var user User
	err = Database.Db.QueryRow(
		"SELECT id, name, pennies FROM users WHERE id = ? AND deleted_at IS NULL",
		id,
	).Scan(&user.ID, &user.Name, &user.Pennies)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond.Error(c, http.StatusNotFound, api.CodeNotFound, "user not found", map[string]any{"id": id})
			return
		}
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to fetch user", nil)
		return
	}
	respond.OK(c, user)
}

func CreateUser(c *gin.Context) {
	var input userPayload
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid payload", map[string]any{"field": "body"})
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, "name is required", map[string]any{"field": "name"})
		return
	}

	res, err := Database.Db.Exec("INSERT INTO users (name, pennies) VALUES (?, ?)", input.Name, input.Pennies)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to create user", nil)
		return
	}
	id, _ := res.LastInsertId()
	respond.JSONOK(c, http.StatusCreated, User{ID: int(id), Name: input.Name, Pennies: input.Pennies})
}

func UpdateUser(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, err.Error(), map[string]any{"field": "id"})
		return
	}

	var input userPayload
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid payload", map[string]any{"field": "body"})
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, "name is required", map[string]any{"field": "name"})
		return
	}

	res, err := Database.Db.Exec(
		"UPDATE users SET name = ?, pennies = ? WHERE id = ? AND deleted_at IS NULL",
		input.Name, input.Pennies, id,
	)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to update user", nil)
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		respond.Error(c, http.StatusNotFound, api.CodeNotFound, "user not found", map[string]any{"id": id})
		return
	}
	respond.OK(c, User{ID: id, Name: input.Name, Pennies: input.Pennies})
}

func DeleteUser(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, err.Error(), map[string]any{"field": "id"})
		return
	}
	tx, err := Database.Db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to delete user", nil)
		return
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(
		c.Request.Context(),
		"UPDATE users SET deleted_at = UTC_TIMESTAMP() WHERE id = ? AND deleted_at IS NULL",
		id,
	)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to delete user", nil)
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		var existing int
		err := tx.QueryRowContext(c.Request.Context(), "SELECT id FROM users WHERE id = ?", id).Scan(&existing)
		if errors.Is(err, sql.ErrNoRows) {
			respond.Error(c, http.StatusNotFound, api.CodeNotFound, "user not found", map[string]any{"id": id})
			return
		}
		if err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to resolve user state", nil)
			return
		}
		respond.NoContent(c)
		return
	}

	if _, err := tx.ExecContext(c.Request.Context(),
		`UPDATE auth_sessions SET revoked_at = UTC_TIMESTAMP() WHERE user_id = ? AND revoked_at IS NULL`, id,
	); err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to revoke user sessions", nil)
		return
	}

	if _, err := tx.ExecContext(c.Request.Context(),
		`UPDATE operator_accounts SET status = 'suspended' WHERE linked_user_id = ?`, id,
	); err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to suspend operator account", nil)
		return
	}

	if err := tx.Commit(); err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to finalize user deletion", nil)
		return
	}

	if AuthStore != nil {
		if actorID, ok := middleware.AuthUserIDFromContext(c); ok {
			reason := strings.TrimSpace(c.GetHeader(middleware.PlatformActionReasonHeader))
			if reason == "" {
				if v, ok := c.Get("platform_action_reason"); ok {
					if s, ok := v.(string); ok {
						reason = strings.TrimSpace(s)
					}
				}
			}
			_ = AuthStore.InsertAdminAuditEvent(
				c.Request.Context(),
				&actorID,
				c.GetString("auth_subject"),
				"user.soft_deleted",
				"user",
				strconv.Itoa(id),
				reason,
				requestid.FromContext(c),
				c.ClientIP(),
				c.GetHeader("User-Agent"),
				nil,
			)
		}
	}
	respond.NoContent(c)
}
