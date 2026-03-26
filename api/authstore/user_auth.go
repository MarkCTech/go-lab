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
		`SELECT password_hash
		 FROM operator_accounts oa
		 INNER JOIN users u ON u.id = oa.linked_user_id
		 WHERE oa.linked_user_id = ?
		   AND oa.status = 'active'
		   AND u.deleted_at IS NULL`,
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
		`SELECT oa.linked_user_id, oa.password_hash
		 FROM operator_accounts oa
		 INNER JOIN users u ON u.id = oa.linked_user_id
		 WHERE oa.email = ?
		   AND oa.status = 'active'
		   AND u.deleted_at IS NULL`,
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO users (name, pennies) VALUES (?, 0)`,
		name,
	)
	if err != nil {
		return 0, err
	}
	uid, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO operator_accounts (linked_user_id, email, password_hash, status)
		 VALUES (?, ?, ?, 'active')`,
		uid, email, passwordHash,
	)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return uid, nil
}

// UpdateUserPasswordHash sets password_hash for a user (caller supplies Argon2id-encoded hash).
func (s *Store) UpdateUserPasswordHash(ctx context.Context, userID int, passwordHash string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE operator_accounts oa
		 INNER JOIN users u ON u.id = oa.linked_user_id
		 SET oa.password_hash = ?
		 WHERE oa.linked_user_id = ? AND u.deleted_at IS NULL`,
		passwordHash, userID,
	)
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
