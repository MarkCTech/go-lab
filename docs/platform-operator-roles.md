# Platform operator roles (Phase A)

**Canonical source of truth:** rows in **`user_platform_roles`** referencing **`platform_roles`**. After migration `000005_*`, seed roles exist: `operator`, `support`, `security_admin`.

**Related:** [platform-control-plane.md](platform-control-plane.md) (domain boundaries + RBAC matrix + route map) · [platform-admin-ui.md](platform-admin-ui.md) · [openapi.yaml](openapi.yaml) · [migrations.md](migrations.md)

## Grant the first operator (SQL)

Replace `YOUR_USER_ID` with the local `users.id` (often `1` for the first registered account).

```sql
INSERT INTO user_platform_roles (user_id, role_id)
SELECT ?, id FROM platform_roles WHERE name = 'operator' LIMIT 1
ON DUPLICATE KEY UPDATE user_id = user_id;
```

MySQL client example:

```bash
mysql -h 127.0.0.1 -u root -p suite_platform -e "INSERT INTO user_platform_roles (user_id, role_id) SELECT 1, id FROM platform_roles WHERE name = 'operator' LIMIT 1 ON DUPLICATE KEY UPDATE user_id = user_id;"
```

Other roles:

```sql
-- support (read-heavy + support ack)
INSERT INTO user_platform_roles (user_id, role_id)
SELECT 2, id FROM platform_roles WHERE name = 'support' LIMIT 1
ON DUPLICATE KEY UPDATE user_id = user_id;

-- security_admin
INSERT INTO user_platform_roles (user_id, role_id)
SELECT 3, id FROM platform_roles WHERE name = 'security_admin' LIMIT 1
ON DUPLICATE KEY UPDATE user_id = user_id;
```
