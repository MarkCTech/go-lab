// Package platformrbac maps platform operator roles to permissions (Phase A control plane).
package platformrbac

// Permission constants for route-level checks.
const (
	PermPlayersRead    = "players.read"
	PermCharactersRead = "characters.read"
	PermBackupsRead    = "backups.read"
	PermSecurityRead   = "security.read"
	PermSecurityWrite  = "security.write"
	PermAuditRead      = "audit.read"
	PermAuditWrite     = "audit.write"
	PermSupportAck     = "platform.support.ack"
)

var rolePermissions = map[string][]string{
	"operator":       {"*"},
	"support":        {PermPlayersRead, PermCharactersRead, PermBackupsRead, PermAuditRead, PermSupportAck, PermSecurityRead},
	"security_admin": {PermSecurityRead, PermSecurityWrite, PermAuditRead, PermAuditWrite, PermPlayersRead},
}

// PermissionsForRole returns permissions granted to a role name (empty if unknown).
func PermissionsForRole(role string) []string {
	return rolePermissions[role]
}

// HasPermission returns true if any role grants the permission (or wildcard *).
func HasPermission(roles []string, required string) bool {
	for _, r := range roles {
		for _, p := range PermissionsForRole(r) {
			if p == "*" || p == required {
				return true
			}
		}
	}
	return false
}
