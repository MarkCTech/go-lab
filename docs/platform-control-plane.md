# Platform control plane - boundaries and RBAC

Audience: operators and implementers working on privileged `/api/v1` surfaces.

## Canonical references

- Permissions source: [`api/platformrbac/permissions.go`](../api/platformrbac/permissions.go)
- Route contract: [openapi.yaml](openapi.yaml)
- Role assignment SQL: [platform-operator-roles.md](platform-operator-roles.md)

## Domain boundaries

| Domain | In platform today | Out of scope for platform authority |
|--------|--------------------|-------------------------------------|
| Identity/auth | `users`, `user_identities`, sessions, invite onboarding | Gameplay authority and simulation state |
| Operator governance | Cases, backup/restore governance, admin audit | Physical backup/restore execution |
| Economy (operator view) | Read-only ledger slice | Authoritative gameplay economy |
| Players/characters | Read endpoints currently stubs | Full gameplay profile/character ownership in Marble |

## Role model

- Effective role model is operator-account based (`operator_accounts` + `operator_account_roles`).
- Compatibility fallback for legacy `user_platform_roles` exists in the store layer only.
- Unknown role names grant no permissions.

## Permission intent (compact)

| Permission family | Typical use |
|------------------|-------------|
| `players.read`, `characters.read` | Operator read surfaces |
| `cases.*`, `sanctions.write`, `recovery.write`, `appeals.resolve` | Case workflow |
| `backups.read`, `backups.restore.*` | Restore governance workflow |
| `economy.read` | Economy operator ledger |
| `audit.read`, `security.read` | Security/audit views |
| `platform.support.ack` | Privileged support acknowledgement |

Role matrix and exact grants are defined in code (`permissions.go`) and should be treated as canonical.

## Enforcement rules

- Privileged control-plane routes use `BearerOrSession`, `RequireHumanUser`, and permission checks.
- Mutating privileged routes require `X-Platform-Action-Reason`.
- `client:*` machine subjects do not satisfy human-user-only route requirements.

## Current shipped surfaces

- Backup restore governance routes under `/api/v1/backups/*`
- Security and audit views (`/api/v1/security/me`, `/api/v1/audit/admin-events`)
- Economy ledger read (`/api/v1/economy/ledger`)
- Operator cases workflow under `/api/v1/cases/*`
- Players/characters read stubs

## Operational guidance

- Prefer narrow role grants over broad `operator` role.
- Keep role changes time-bound and auditable.
- Review admin audit events after high-risk operations.

## Related

- [split-host-operations.md](split-host-operations.md)
- [data-ownership.md](data-ownership.md)
- [ops-secret-rotation.md](ops-secret-rotation.md)
