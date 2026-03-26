package authstore

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var (
	ErrOperatorInviteNotFound = errors.New("operator invite not found")
	ErrOperatorInviteUsed     = errors.New("operator invite already used")
	ErrOperatorInviteExpired  = errors.New("operator invite expired")
	ErrOperatorInviteEmailMismatch = errors.New("operator invite email mismatch")
	ErrOperatorRoleNotFound   = errors.New("operator role not found")
	ErrOperatorEmailExists    = errors.New("operator email already exists")
)

type OperatorInviteResult struct {
	LinkedUserID int
	Email        string
	RoleName     string
}

func (s *Store) CreateOperatorInvite(
	ctx context.Context,
	tokenHash, email, displayName, roleName string,
	linkedUserID, invitedByUserID *int,
	expiresAt time.Time,
	metaJSON []byte,
) error {
	if s == nil || s.db == nil {
		return errors.New("nil db")
	}
	var linked any
	if linkedUserID != nil {
		linked = *linkedUserID
	}
	var invitedBy any
	if invitedByUserID != nil {
		invitedBy = *invitedByUserID
	}
	var meta any
	if len(metaJSON) > 0 {
		meta = metaJSON
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO operator_invites
		 (token_hash, email, display_name, role_name, linked_user_id, invited_by_user_id, expires_at, meta_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		tokenHash, email, displayName, roleName, linked, invitedBy, expiresAt.UTC(), meta,
	)
	return err
}

func (s *Store) AcceptOperatorInvite(
	ctx context.Context,
	tokenHash, inputEmail, passwordHash, overrideDisplayName string,
) (OperatorInviteResult, error) {
	if s == nil || s.db == nil {
		return OperatorInviteResult{}, errors.New("nil db")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return OperatorInviteResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var inviteID int64
	var email, invitedDisplayName, roleName string
	var linked sql.NullInt64
	var invitedBy sql.NullInt64
	var expiresAt time.Time
	var consumedAt sql.NullTime
	err = tx.QueryRowContext(ctx,
		`SELECT id, email, display_name, role_name, linked_user_id, invited_by_user_id, expires_at, consumed_at
		 FROM operator_invites
		 WHERE token_hash = ?
		 FOR UPDATE`,
		tokenHash,
	).Scan(&inviteID, &email, &invitedDisplayName, &roleName, &linked, &invitedBy, &expiresAt, &consumedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return OperatorInviteResult{}, ErrOperatorInviteNotFound
	}
	if err != nil {
		return OperatorInviteResult{}, err
	}
	if consumedAt.Valid {
		return OperatorInviteResult{}, ErrOperatorInviteUsed
	}
	if !time.Now().UTC().Before(expiresAt.UTC()) {
		return OperatorInviteResult{}, ErrOperatorInviteExpired
	}
	if !strings.EqualFold(strings.TrimSpace(email), strings.TrimSpace(inputEmail)) {
		return OperatorInviteResult{}, ErrOperatorInviteEmailMismatch
	}

	displayName := strings.TrimSpace(overrideDisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(invitedDisplayName)
	}
	if displayName == "" {
		displayName = "Operator"
	}
	if len(displayName) > 100 {
		displayName = displayName[:100]
	}

	var roleID uint16
	err = tx.QueryRowContext(ctx, `SELECT id FROM platform_roles WHERE name = ?`, roleName).Scan(&roleID)
	if errors.Is(err, sql.ErrNoRows) {
		return OperatorInviteResult{}, ErrOperatorRoleNotFound
	}
	if err != nil {
		return OperatorInviteResult{}, err
	}

	var linkedUserID int64
	if linked.Valid && linked.Int64 > 0 {
		linkedUserID = linked.Int64
	} else {
		res, err := tx.ExecContext(ctx, `INSERT INTO users (name, pennies) VALUES (?, 0)`, displayName)
		if err != nil {
			return OperatorInviteResult{}, err
		}
		linkedUserID, err = res.LastInsertId()
		if err != nil {
			return OperatorInviteResult{}, err
		}
	}

	var exists int
	err = tx.QueryRowContext(ctx,
		`SELECT 1 FROM operator_accounts WHERE email = ? OR linked_user_id = ? LIMIT 1`,
		email, linkedUserID,
	).Scan(&exists)
	if err == nil {
		return OperatorInviteResult{}, ErrOperatorEmailExists
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return OperatorInviteResult{}, err
	}

	var invitedByAny any
	if invitedBy.Valid && invitedBy.Int64 > 0 {
		invitedByAny = invitedBy.Int64
	}

	res, err := tx.ExecContext(ctx,
		`INSERT INTO operator_accounts
		 (linked_user_id, email, password_hash, status, invited_by_user_id, last_login_at)
		 VALUES (?, ?, ?, 'active', ?, NULL)`,
		linkedUserID, email, passwordHash, invitedByAny,
	)
	if err != nil {
		return OperatorInviteResult{}, err
	}
	operatorAccountID, err := res.LastInsertId()
	if err != nil {
		return OperatorInviteResult{}, err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO operator_account_roles (operator_account_id, role_id) VALUES (?, ?)`,
		operatorAccountID, roleID,
	)
	if err != nil {
		return OperatorInviteResult{}, err
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE operator_invites SET consumed_at = UTC_TIMESTAMP() WHERE id = ? AND consumed_at IS NULL`,
		inviteID,
	)
	if err != nil {
		return OperatorInviteResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return OperatorInviteResult{}, err
	}
	return OperatorInviteResult{
		LinkedUserID: int(linkedUserID),
		Email:        email,
		RoleName:     roleName,
	}, nil
}

func (s *Store) TouchOperatorLoginByEmail(ctx context.Context, email string) error {
	if s == nil || s.db == nil {
		return errors.New("nil db")
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE operator_accounts SET last_login_at = UTC_TIMESTAMP() WHERE email = ?`,
		email,
	)
	return err
}
