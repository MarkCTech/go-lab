# Platform operator roles

Role assignments are now operator-account based.

## Canonical model

- Roles: `platform_roles`
- Operator identities: `operator_accounts` (linked to `users.id`)
- Role links: `operator_account_roles`
- Invite onboarding: `operator_invites`

This model is introduced in migration `000009_*`.

## Seeded roles

- `operator` - break-glass full access (`*`)
- `support` - read-heavy support + restore request + case operations
- `security_admin` - security/governance operations including restore approval/fulfillment
- `gm_liveops` - player/character case workflow operations

Permission details and route mapping live in [platform-control-plane.md](platform-control-plane.md) and [`api/platformrbac/permissions.go`](../api/platformrbac/permissions.go).

## Grant role to an operator account

```sql
-- grant operator to operator account id 1
INSERT INTO operator_account_roles (operator_account_id, role_id)
SELECT 1, id FROM platform_roles WHERE name = 'operator' LIMIT 1
ON DUPLICATE KEY UPDATE operator_account_id = operator_account_id;
```

```sql
-- support
INSERT INTO operator_account_roles (operator_account_id, role_id)
SELECT 2, id FROM platform_roles WHERE name = 'support' LIMIT 1
ON DUPLICATE KEY UPDATE operator_account_id = operator_account_id;

-- security_admin
INSERT INTO operator_account_roles (operator_account_id, role_id)
SELECT 3, id FROM platform_roles WHERE name = 'security_admin' LIMIT 1
ON DUPLICATE KEY UPDATE operator_account_id = operator_account_id;

-- gm_liveops
INSERT INTO operator_account_roles (operator_account_id, role_id)
SELECT 4, id FROM platform_roles WHERE name = 'gm_liveops' LIMIT 1
ON DUPLICATE KEY UPDATE operator_account_id = operator_account_id;
```

## Legacy compatibility note

`api/authstore/platform_rbac.go` includes a fallback path that can read `user_platform_roles` when operator-account role tables are unavailable. Treat that as compatibility behavior, not the target operating model.

## Related

- [platform-control-plane.md](platform-control-plane.md)
- [platform-admin-ui.md](platform-admin-ui.md)
- [migrations.md](migrations.md)
- [openapi.yaml](openapi.yaml)
