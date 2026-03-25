# Platform control plane — Phase A reference

**Audience:** operators and implementers. **Companion:** [data-ownership.md](data-ownership.md) (suite-wide ownership), [platform-operator-roles.md](platform-operator-roles.md) (SQL to grant roles).

This doc **closes Phase A planning gaps**: domain boundaries (what exists in go-lab today) and the **RBAC matrix** (must match [`api/platformrbac/permissions.go`](../api/platformrbac/permissions.go) — update both when adding roles or permissions).

---

## 1. Domain boundaries (Phase A)

| Domain | In go-lab DB / API today | Authoritative / next step |
|--------|---------------------------|---------------------------|
| **identity_*** | `users`, `user_identities`, auth session + desktop exchange tables | Platform; see migrations `000002`–`000004` |
| **player_*** | No player tables; `GET /api/v1/players` stub | Gameplay-linked profiles live in **Marble** / suite DBs keyed by `platform_user_id` ([data-ownership.md](data-ownership.md)) |
| **character_*** | No character tables; `GET /api/v1/characters` stub | **Marble**-authoritative |
| **session_*** (operator sense) | Login = `auth_sessions`; join/desktop = `000004_*` | Extended “session registry” UI/API = Phase B if needed |
| **backup_*** | `GET /api/v1/backups/status` stub only | Policy/run/restore tables + workflows = **Phase C** |
| **audit_*** (control plane) | `admin_audit_events` (immutable rows for privileged actions) | Extend actions as Phase B/C routes ship |
| **audit_*** (auth) | `auth_audit_events` | Existing auth security trail |

---

## 2. RBAC matrix (roles × permissions)

Permissions are string constants in [`api/platformrbac/permissions.go`](../api/platformrbac/permissions.go). **`operator`** grants **`*`** (wildcard). **§3** lists routes that call `RequirePlatformPermission` today; `security.write` and `audit.write` are granted to `security_admin` but **no Phase A HTTP route checks them yet** (they may still appear in `GET /api/v1/security/me`).

| Permission | `operator` | `support` | `security_admin` |
|------------|:----------:|:---------:|:----------------:|
| `players.read` | yes | yes | yes |
| `characters.read` | yes | yes | no |
| `backups.read` | yes | yes | no |
| `security.read` | yes | yes | yes |
| `security.write` | yes | no | yes |
| `audit.read` | yes | yes | yes |
| `audit.write` | yes | no | yes |
| `platform.support.ack` | yes | yes | no |

Unknown role names grant **no** permissions.

---

## 3. Routes ↔ required permission

All routes below use **`BearerOrSession` + `RequireHumanUser`** (subjects must be `user:<id>`; `client:*` cannot pass).

| Method | Path | Permission |
|--------|------|------------|
| GET | `/api/v1/players` | `players.read` |
| GET | `/api/v1/characters` | `characters.read` |
| GET | `/api/v1/backups/status` | `backups.read` |
| GET | `/api/v1/security/me` | `security.read` |
| GET | `/api/v1/audit/admin-events` | `audit.read` |
| POST | `/api/v1/support/ack` | `platform.support.ack` + header `X-Platform-Action-Reason` (min length) |

Contract: [openapi.yaml](openapi.yaml) (`platform` tag).

---

## 4. Granting roles

See [platform-operator-roles.md](platform-operator-roles.md) for `INSERT` examples into `user_platform_roles`.

---

## Phase B/C (not Phase A)

- Rich player/character **data** and **mutations** (sanctions, recovery, etc.).
- Backup **policy/run/restore** schema and approval flows.
- **Role assignment API/UI** (today: SQL only).
- Unified **audit taxonomy** across `auth_audit_events` and `admin_audit_events` ([MASTER_PLAN.md](MASTER_PLAN.md) §9 P2).
