-- Identity/session domain: user auth columns, opaque session cookies, refresh-token storage (rotation-ready), audit trail.
-- Builds on 000001 `users` table. Application may not use every table in v1; schema is migration-only.

ALTER TABLE users
  ADD COLUMN email VARCHAR(255) NULL DEFAULT NULL AFTER name,
  ADD COLUMN password_hash VARCHAR(255) NULL DEFAULT NULL AFTER email,
  ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ADD COLUMN updated_at TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP;

CREATE UNIQUE INDEX idx_users_email ON users (email);

CREATE TABLE IF NOT EXISTS auth_sessions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  token_hash CHAR(64) NOT NULL COMMENT 'SHA-256 hex of opaque session token',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_seen_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NOT NULL,
  absolute_expires_at TIMESTAMP NOT NULL,
  revoked_at TIMESTAMP NULL DEFAULT NULL,
  ip VARCHAR(45) NULL,
  user_agent VARCHAR(512) NULL,
  CONSTRAINT fk_auth_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  UNIQUE KEY uq_auth_sessions_token_hash (token_hash),
  KEY idx_auth_sessions_user (user_id),
  KEY idx_auth_sessions_revoked (revoked_at)
);

-- Reserved for future refresh-token / key-rotation flows; browser v1 uses session cookie + POST /auth/refresh sliding window only.
CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  session_id BIGINT UNSIGNED NOT NULL,
  token_hash CHAR(64) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NOT NULL,
  revoked_at TIMESTAMP NULL DEFAULT NULL,
  replaced_by_id BIGINT UNSIGNED NULL,
  CONSTRAINT fk_auth_refresh_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_auth_refresh_session FOREIGN KEY (session_id) REFERENCES auth_sessions(id) ON DELETE CASCADE,
  UNIQUE KEY uq_auth_refresh_token_hash (token_hash),
  KEY idx_auth_refresh_session (session_id)
);

CREATE TABLE IF NOT EXISTS auth_audit_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  event_type VARCHAR(64) NOT NULL,
  user_id INT NULL,
  ip VARCHAR(45) NULL,
  user_agent VARCHAR(512) NULL,
  subject_hint VARCHAR(255) NULL,
  meta_json JSON NULL,
  KEY idx_auth_audit_created (created_at),
  KEY idx_auth_audit_user (user_id),
  KEY idx_auth_audit_type (event_type)
);
