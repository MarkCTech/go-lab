package myhandlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/codemarked/go-lab/api/api"
	"github.com/codemarked/go-lab/api/respond"
	"github.com/gin-gonic/gin"
)

type userPayload struct {
	Name    string `json:"name" binding:"required,min=1,max=100"`
	Pennies int    `json:"pennies" binding:"gte=0"`
}

func GetUsers(c *gin.Context) {
	rows, err := Database.Db.Query("SELECT id, name, pennies FROM users ORDER BY pennies DESC, id ASC")
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
		"SELECT id, name, pennies FROM users WHERE name LIKE CONCAT('%', ?, '%') ORDER BY pennies DESC, id ASC",
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
	err = Database.Db.QueryRow("SELECT id, name, pennies FROM users WHERE id = ?", id).Scan(&user.ID, &user.Name, &user.Pennies)
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

	res, err := Database.Db.Exec("UPDATE users SET name = ?, pennies = ? WHERE id = ?", input.Name, input.Pennies, id)
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
	res, err := Database.Db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to delete user", nil)
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		respond.Error(c, http.StatusNotFound, api.CodeNotFound, "user not found", map[string]any{"id": id})
		return
	}
	respond.NoContent(c)
}
