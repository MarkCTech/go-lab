package middleware

import (
	"net/http"
	"slices"

	"github.com/codemarked/go-lab/api/api"
	"github.com/codemarked/go-lab/api/authstore"
	"github.com/codemarked/go-lab/api/platformrbac"
	"github.com/gin-gonic/gin"
)

// RequirePlatformPermission enforces a human user subject with a role granting the permission
// from user_platform_roles → platform_roles (see docs/platform-operator-roles.md).
func RequirePlatformPermission(store *authstore.Store, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, ok := AuthUserIDFromContext(c)
		if !ok {
			writeAuthError(c, http.StatusForbidden, api.CodeForbidden, "this operation requires a signed-in user account")
			c.Abort()
			return
		}
		roles, err := effectivePlatformRoles(c, store, uid)
		if err != nil {
			writeAuthError(c, http.StatusInternalServerError, api.CodeInternal, "failed to resolve platform roles")
			c.Abort()
			return
		}
		if !platformrbac.HasPermission(roles, permission) {
			writeAuthError(c, http.StatusForbidden, api.CodeForbidden, "insufficient platform permissions for this operation")
			c.Abort()
			return
		}
		c.Set("platform_roles", roles)
		c.Next()
	}
}

func effectivePlatformRoles(c *gin.Context, store *authstore.Store, userID int) ([]string, error) {
	var roles []string
	if store != nil {
		dbRoles, err := store.ListPlatformRoleNamesForUser(c.Request.Context(), userID)
		if err != nil {
			return nil, err
		}
		roles = append(roles, dbRoles...)
	}
	slices.Sort(roles)
	roles = slices.Compact(roles)
	return roles, nil
}
