DROP TABLE IF EXISTS auth_audit_events;
DROP TABLE IF EXISTS auth_refresh_tokens;
DROP TABLE IF EXISTS auth_sessions;

DROP INDEX idx_users_email ON users;

ALTER TABLE users
  DROP COLUMN email,
  DROP COLUMN password_hash,
  DROP COLUMN created_at,
  DROP COLUMN updated_at;
