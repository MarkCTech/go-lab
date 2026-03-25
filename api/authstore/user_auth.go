package authstore

import (
	"context"
	"database/sql"
	"errors"
)

// GetUserPasswordHashByID returns the stored password hash for the user (empty if unset).
func (s *Store) GetUserPasswordHashByID(ctx context.Context, userID int) (passwordHash string, err error) {
	var hash sql.NullString
	err = s.db.QueryRowContext(ctx,
		`SELECT password_hash FROM users WHERE id = ?`,
		userID,
	).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", sql.ErrNoRows
	}
	if err != nil {
		return "", err
	}
	if !hash.Valid {
		return "", nil
	}
	return hash.String, nil
}

// GetUserAuthByEmail returns user id and password hash for login (hash may be empty for legacy rows).
func (s *Store) GetUserAuthByEmail(ctx context.Context, email string) (id int, passwordHash string, err error) {
	var hash sql.NullString
	err = s.db.QueryRowContext(ctx,
		`SELECT id, password_hash FROM users WHERE email = ?`,
		email,
	).Scan(&id, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", sql.ErrNoRows
	}
	if err != nil {
		return 0, "", err
	}
	if !hash.Valid {
		return id, "", nil
	}
	return id, hash.String, nil
}

// CreateRegisteredUser inserts a user row with credentials (email unique enforced by DB).
func (s *Store) CreateRegisteredUser(ctx context.Context, email, name, passwordHash string) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (name, email, password_hash, pennies) VALUES (?, ?, ?, 0)`,
		name, email, passwordHash,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateUserPasswordHash sets password_hash for a user (caller supplies Argon2id-encoded hash).
func (s *Store) UpdateUserPasswordHash(ctx context.Context, userID int, passwordHash string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// RevokeAllSessionsForUser marks every active session revoked (password-change / admin flows; not used on single logout).
func (s *Store) RevokeAllSessionsForUser(ctx context.Context, userID int) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE auth_sessions SET revoked_at = UTC_TIMESTAMP() WHERE user_id = ? AND revoked_at IS NULL`,
		userID,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
