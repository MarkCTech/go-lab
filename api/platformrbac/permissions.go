// Package platformrbac maps platform operator roles to permissions (platform control plane).
package platformrbac

// Permission constants for route-level checks.
const (
	PermPlayersRead             = "players.read"
	PermCharactersRead          = "characters.read"
	PermBackupsRead             = "backups.read"
	PermBackupsRestoreRequest   = "backups.restore.request"
	PermBackupsRestoreApprove   = "backups.restore.approve"
	PermBackupsRestoreFulfill   = "backups.restore.fulfill"
	PermSecurityRead            = "security.read"
	PermSecurityWrite           = "security.write"
	PermAuditRead               = "audit.read"
	PermAuditWrite              = "audit.write"
	PermSupportAck              = "platform.support.ack"
	PermEconomyRead             = "economy.read"
	PermUsersDelete             = "users.delete"
	// Operator cases: player/character workflows (cases, sanctions, recovery, appeals).
	PermCasesRead      = "cases.read"
	PermCasesWrite     = "cases.write"
	PermSanctionsWrite = "sanctions.write"
	PermRecoveryWrite  = "recovery.write"
	PermAppealsResolve = "appeals.resolve"
)

var rolePermissions = map[string][]string{
	"operator": {"*"},
	"support": {
		PermPlayersRead, PermCharactersRead, PermBackupsRead, PermBackupsRestoreRequest,
		PermEconomyRead, PermAuditRead, PermSupportAck, PermSecurityRead,
		PermCasesRead, PermCasesWrite, PermRecoveryWrite, PermAppealsResolve,
	},
	"security_admin": {
		PermSecurityRead, PermSecurityWrite, PermAuditRead, PermAuditWrite,
		PermPlayersRead, PermEconomyRead, PermBackupsRead,
		PermBackupsRestoreApprove, PermBackupsRestoreFulfill,
		PermCasesRead, PermSanctionsWrite, PermAppealsResolve, PermUsersDelete,
	},
	// GM / live-ops: full player-character workflow slice; grant via SQL after migration 000008.
	"gm_liveops": {
		PermPlayersRead, PermCharactersRead,
		PermCasesRead, PermCasesWrite, PermSanctionsWrite, PermRecoveryWrite, PermAppealsResolve,
		PermAuditRead, PermSupportAck, PermSecurityRead, PermEconomyRead,
	},
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
