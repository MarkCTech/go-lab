-- Phase A: platform operator RBAC + immutable admin audit trail (control-plane foundations).
-- Assign roles via user_platform_roles → platform_roles (see docs/platform-operator-roles.md).

CREATE TABLE IF NOT EXISTS platform_roles (
  id SMALLINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_platform_roles_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS user_platform_roles (
  user_id INT NOT NULL,
  role_id SMALLINT UNSIGNED NOT NULL,
  assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, role_id),
  KEY idx_user_platform_roles_role (role_id),
  CONSTRAINT fk_user_platform_roles_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_user_platform_roles_role FOREIGN KEY (role_id) REFERENCES platform_roles (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS admin_audit_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  actor_user_id INT DEFAULT NULL,
  auth_subject VARCHAR(255) NOT NULL,
  action VARCHAR(128) NOT NULL,
  resource_type VARCHAR(64) NOT NULL,
  resource_id VARCHAR(128) DEFAULT NULL,
  reason TEXT,
  request_id VARCHAR(128) DEFAULT NULL,
  ip VARCHAR(45) DEFAULT NULL,
  user_agent VARCHAR(512) DEFAULT NULL,
  meta_json JSON DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_admin_audit_created (created_at),
  KEY idx_admin_audit_actor (actor_user_id),
  KEY idx_admin_audit_action (action),
  CONSTRAINT fk_admin_audit_actor FOREIGN KEY (actor_user_id) REFERENCES users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO platform_roles (name) VALUES ('operator'), ('support'), ('security_admin');
