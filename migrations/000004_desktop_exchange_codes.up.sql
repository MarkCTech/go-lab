-- One-time code bridge for desktop login handoff (browser-authenticated user -> desktop bearer token).
CREATE TABLE IF NOT EXISTS auth_desktop_exchange_codes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  code_hash CHAR(64) NOT NULL COMMENT 'SHA-256 hex of one-time desktop exchange code',
  code_challenge VARCHAR(128) NOT NULL COMMENT 'PKCE-style S256 challenge for code_verifier proof',
  session_id VARCHAR(128) NOT NULL,
  callback_uri VARCHAR(512) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NOT NULL,
  consumed_at TIMESTAMP NULL DEFAULT NULL,
  CONSTRAINT fk_auth_desktop_exchange_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  UNIQUE KEY uq_auth_desktop_exchange_code_hash (code_hash),
  KEY idx_auth_desktop_exchange_user (user_id),
  KEY idx_auth_desktop_exchange_expires (expires_at),
  KEY idx_auth_desktop_exchange_consumed (consumed_at)
) ENGINE=InnoDB;
