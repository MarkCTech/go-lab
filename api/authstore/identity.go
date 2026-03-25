package authstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// LocalUserIDForIdentity returns the local users.id for an external (issuer, subject) pair, if any.
func (s *Store) LocalUserIDForIdentity(ctx context.Context, issuer, subject string) (int64, bool, error) {
	if s.db == nil {
		return 0, false, errors.New("nil db")
	}
	var uid int64
	err := s.db.QueryRowContext(ctx,
		`SELECT user_id FROM user_identities WHERE issuer = ? AND subject = ?`,
		issuer, subject,
	).Scan(&uid)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return uid, true, nil
}

// EnsureOIDCUser returns the local user id for (issuer, subject), creating a minimal user row on first sight.
func (s *Store) EnsureOIDCUser(ctx context.Context, issuer, subject, displayName string) (int64, error) {
	if s.db == nil {
		return 0, errors.New("nil db")
	}
	if issuer == "" || subject == "" {
		return 0, errors.New("issuer and subject required")
	}
	name := displayName
	if name == "" {
		name = "OIDC user"
	}
	if len(name) > 100 {
		name = name[:100]
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	var uid int64
	err = tx.QueryRowContext(ctx,
		`SELECT user_id FROM user_identities WHERE issuer = ? AND subject = ?`,
		issuer, subject,
	).Scan(&uid)
	if err == nil {
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return uid, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	res, err := tx.ExecContext(ctx,
		`INSERT INTO users (name, email, password_hash, pennies) VALUES (?, NULL, NULL, 0)`,
		name,
	)
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO user_identities (user_id, issuer, subject) VALUES (?, ?, ?)`,
		id, issuer, subject,
	)
	if err != nil {
		return 0, fmt.Errorf("insert identity: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}
