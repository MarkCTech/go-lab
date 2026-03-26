-- Domain users soft-delete support.
ALTER TABLE users
  ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL AFTER updated_at;

CREATE INDEX idx_users_deleted_at ON users (deleted_at);
