-- Separate operator authentication identities from domain users.
-- Keeps users table for domain/business entities while introducing dedicated operator auth tables.

CREATE TABLE IF NOT EXISTS operator_accounts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  linked_user_id INT NOT NULL,
  email VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  invited_by_user_id INT NULL,
  last_login_at TIMESTAMP NULL DEFAULT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_operator_accounts_linked_user (linked_user_id),
  UNIQUE KEY uq_operator_accounts_email (email),
  KEY idx_operator_accounts_status (status),
  KEY idx_operator_accounts_invited_by (invited_by_user_id),
  CONSTRAINT fk_operator_accounts_linked_user FOREIGN KEY (linked_user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_operator_accounts_invited_by FOREIGN KEY (invited_by_user_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS operator_account_roles (
  operator_account_id BIGINT UNSIGNED NOT NULL,
  role_id SMALLINT UNSIGNED NOT NULL,
  assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (operator_account_id, role_id),
  KEY idx_operator_account_roles_role (role_id),
  CONSTRAINT fk_operator_account_roles_account FOREIGN KEY (operator_account_id) REFERENCES operator_accounts(id) ON DELETE CASCADE,
  CONSTRAINT fk_operator_account_roles_role FOREIGN KEY (role_id) REFERENCES platform_roles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS operator_invites (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  token_hash CHAR(64) NOT NULL COMMENT 'SHA-256 hex of one-time invite token',
  email VARCHAR(255) NOT NULL,
  display_name VARCHAR(100) NOT NULL,
  role_name VARCHAR(64) NOT NULL,
  linked_user_id INT NULL,
  invited_by_user_id INT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NOT NULL,
  consumed_at TIMESTAMP NULL DEFAULT NULL,
  meta_json JSON NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_operator_invites_token_hash (token_hash),
  KEY idx_operator_invites_email (email),
  KEY idx_operator_invites_expires (expires_at),
  KEY idx_operator_invites_consumed (consumed_at),
  KEY idx_operator_invites_role_name (role_name),
  CONSTRAINT fk_operator_invites_linked_user FOREIGN KEY (linked_user_id) REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT fk_operator_invites_invited_by FOREIGN KEY (invited_by_user_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Backfill: existing login-capable rows in users become operator accounts.
INSERT INTO operator_accounts (linked_user_id, email, password_hash, status, created_at, updated_at)
SELECT u.id, u.email, u.password_hash, 'active', u.created_at, u.updated_at
FROM users u
WHERE u.email IS NOT NULL
  AND u.password_hash IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM operator_accounts oa
    WHERE oa.linked_user_id = u.id OR oa.email = u.email
  );

-- Backfill role assignments from legacy user_platform_roles mappings.
INSERT INTO operator_account_roles (operator_account_id, role_id, assigned_at)
SELECT oa.id, upr.role_id, upr.assigned_at
FROM operator_accounts oa
INNER JOIN user_platform_roles upr ON upr.user_id = oa.linked_user_id
LEFT JOIN operator_account_roles oar
  ON oar.operator_account_id = oa.id
 AND oar.role_id = upr.role_id
WHERE oar.operator_account_id IS NULL;
