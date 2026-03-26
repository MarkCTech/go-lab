package authstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Operator case workflow: platform governance records; Marble applies gameplay effects out of band.

var (
	ErrInvalidCaseStatus   = errors.New("invalid case status")
	ErrInvalidCasePriority = errors.New("invalid case priority")
)

const (
	CaseStatusOpen       = "open"
	CaseStatusInProgress = "in_progress"
	CaseStatusResolved   = "resolved"
	CaseStatusClosed     = "closed"
)

const (
	CasePriorityLow    = "low"
	CasePriorityNormal = "normal"
	CasePriorityHigh   = "high"
	CasePriorityUrgent = "urgent"
)

func isValidCaseStatus(v string) bool {
	switch v {
	case CaseStatusOpen, CaseStatusInProgress, CaseStatusResolved, CaseStatusClosed:
		return true
	default:
		return false
	}
}

func isValidCasePriority(v string) bool {
	switch v {
	case CasePriorityLow, CasePriorityNormal, CasePriorityHigh, CasePriorityUrgent:
		return true
	default:
		return false
	}
}

const (
	CaseActionSanction        = "sanction"
	CaseActionRecoveryRequest = "recovery_request"
	CaseActionAppealResolve   = "appeal_resolve"
)

// OperatorCaseRow is one row from operator_cases.
type OperatorCaseRow struct {
	ID                    int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Status                string
	Priority              string
	SubjectPlatformUserID int
	SubjectCharacterRef   sql.NullString
	Title                 string
	Description           sql.NullString
	CreatedByUserID       int
	AssignedToUserID      sql.NullInt64
}

// OperatorCaseNoteRow is a note on a case.
type OperatorCaseNoteRow struct {
	ID              int64
	CaseID          int64
	CreatedAt       time.Time
	Body            string
	CreatedByUserID int
}

// OperatorCaseActionRow is an append-only action on a case.
type OperatorCaseActionRow struct {
	ID          int64
	CaseID      int64
	CreatedAt   time.Time
	ActionKind  string
	PayloadJSON []byte
	Reason      sql.NullString
	ActorUserID int
}

// ListOperatorCasesQuery filters list results.
type ListOperatorCasesQuery struct {
	Limit         int
	Status        string
	SubjectUserID *int
	BeforeID      *int64
}

// CreateOperatorCase inserts a new case.
func (s *Store) CreateOperatorCase(ctx context.Context, subjectUserID int, characterRef *string, title, description string, priority string, createdBy int) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("nil db")
	}
	title = strings.TrimSpace(title)
	if subjectUserID <= 0 || title == "" || createdBy <= 0 {
		return 0, errors.New("invalid operator case fields")
	}
	if priority == "" {
		priority = CasePriorityNormal
	}
	if !isValidCasePriority(priority) {
		return 0, ErrInvalidCasePriority
	}
	var desc any
	if strings.TrimSpace(description) != "" {
		desc = description
	} else {
		desc = nil
	}
	var cref any
	if characterRef != nil && strings.TrimSpace(*characterRef) != "" {
		cref = strings.TrimSpace(*characterRef)
	} else {
		cref = nil
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO operator_cases (status, priority, subject_platform_user_id, subject_character_ref, title, description, created_by_user_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		CaseStatusOpen, priority, subjectUserID, cref, title, desc, createdBy,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetOperatorCase loads one case by id.
func (s *Store) GetOperatorCase(ctx context.Context, id int64) (OperatorCaseRow, error) {
	if s == nil || s.db == nil {
		return OperatorCaseRow{}, errors.New("nil db")
	}
	var r OperatorCaseRow
	err := s.db.QueryRowContext(ctx,
		`SELECT id, created_at, updated_at, status, priority, subject_platform_user_id, subject_character_ref, title, description, created_by_user_id, assigned_to_user_id
		 FROM operator_cases WHERE id = ?`, id,
	).Scan(
		&r.ID, &r.CreatedAt, &r.UpdatedAt, &r.Status, &r.Priority, &r.SubjectPlatformUserID,
		&r.SubjectCharacterRef, &r.Title, &r.Description, &r.CreatedByUserID, &r.AssignedToUserID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return OperatorCaseRow{}, sql.ErrNoRows
	}
	return r, err
}

// ListOperatorCases returns newest first.
func (s *Store) ListOperatorCases(ctx context.Context, q ListOperatorCasesQuery) ([]OperatorCaseRow, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("nil db")
	}
	limit := q.Limit
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var b strings.Builder
	b.WriteString(`SELECT id, created_at, updated_at, status, priority, subject_platform_user_id, subject_character_ref, title, description, created_by_user_id, assigned_to_user_id
		FROM operator_cases WHERE 1=1`)
	args := make([]any, 0, 6)
	if q.Status != "" {
		b.WriteString(` AND status = ?`)
		args = append(args, q.Status)
	}
	if q.SubjectUserID != nil {
		b.WriteString(` AND subject_platform_user_id = ?`)
		args = append(args, *q.SubjectUserID)
	}
	if q.BeforeID != nil {
		b.WriteString(` AND id < ?`)
		args = append(args, *q.BeforeID)
	}
	b.WriteString(` ORDER BY id DESC LIMIT ?`)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list operator cases: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []OperatorCaseRow
	for rows.Next() {
		var r OperatorCaseRow
		if err := rows.Scan(
			&r.ID, &r.CreatedAt, &r.UpdatedAt, &r.Status, &r.Priority, &r.SubjectPlatformUserID,
			&r.SubjectCharacterRef, &r.Title, &r.Description, &r.CreatedByUserID, &r.AssignedToUserID,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if out == nil {
		out = []OperatorCaseRow{}
	}
	return out, rows.Err()
}

// UpdateOperatorCase updates status, priority, assignment.
func (s *Store) UpdateOperatorCase(ctx context.Context, id int64, status, priority *string, assignedToUserID *int) error {
	if s == nil || s.db == nil {
		return errors.New("nil db")
	}
	if id < 1 {
		return errors.New("invalid case id")
	}
	var sets []string
	var args []any
	if status != nil && strings.TrimSpace(*status) != "" {
		nextStatus := strings.TrimSpace(*status)
		if !isValidCaseStatus(nextStatus) {
			return ErrInvalidCaseStatus
		}
		sets = append(sets, "status = ?")
		args = append(args, nextStatus)
	}
	if priority != nil && strings.TrimSpace(*priority) != "" {
		nextPriority := strings.TrimSpace(*priority)
		if !isValidCasePriority(nextPriority) {
			return ErrInvalidCasePriority
		}
		sets = append(sets, "priority = ?")
		args = append(args, nextPriority)
	}
	if assignedToUserID != nil {
		if *assignedToUserID <= 0 {
			sets = append(sets, "assigned_to_user_id = NULL")
		} else {
			sets = append(sets, "assigned_to_user_id = ?")
			args = append(args, *assignedToUserID)
		}
	}
	if len(sets) == 0 {
		return errors.New("no fields to update")
	}
	args = append(args, id)
	q := `UPDATE operator_cases SET ` + strings.Join(sets, ", ") + ` WHERE id = ?`
	res, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// InsertOperatorCaseNote adds a note.
func (s *Store) InsertOperatorCaseNote(ctx context.Context, caseID int64, body string, createdBy int) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("nil db")
	}
	body = strings.TrimSpace(body)
	if caseID < 1 || body == "" || createdBy <= 0 {
		return 0, errors.New("invalid note")
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO operator_case_notes (case_id, body, created_by_user_id) VALUES (?, ?, ?)`,
		caseID, body, createdBy,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListOperatorCaseNotes returns notes oldest first for a case.
func (s *Store) ListOperatorCaseNotes(ctx context.Context, caseID int64) ([]OperatorCaseNoteRow, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("nil db")
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, case_id, created_at, body, created_by_user_id FROM operator_case_notes WHERE case_id = ? ORDER BY id ASC`,
		caseID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []OperatorCaseNoteRow
	for rows.Next() {
		var r OperatorCaseNoteRow
		if err := rows.Scan(&r.ID, &r.CaseID, &r.CreatedAt, &r.Body, &r.CreatedByUserID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if out == nil {
		out = []OperatorCaseNoteRow{}
	}
	return out, rows.Err()
}

// InsertOperatorCaseAction appends a privileged action row.
func (s *Store) InsertOperatorCaseAction(ctx context.Context, caseID int64, kind string, payload any, reason string, actorUserID int) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("nil db")
	}
	if caseID < 1 || strings.TrimSpace(kind) == "" || actorUserID <= 0 {
		return 0, errors.New("invalid case action")
	}
	var meta []byte
	var err error
	if payload != nil {
		meta, err = json.Marshal(payload)
		if err != nil {
			return 0, err
		}
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO operator_case_actions (case_id, action_kind, payload_json, reason, actor_user_id) VALUES (?, ?, ?, ?, ?)`,
		caseID, strings.TrimSpace(kind), nullBytes(meta), nullStr(reason), actorUserID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func nullBytes(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

// ListOperatorCaseActions returns actions newest first.
func (s *Store) ListOperatorCaseActions(ctx context.Context, caseID int64, limit int) ([]OperatorCaseActionRow, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("nil db")
	}
	if limit < 1 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, case_id, created_at, action_kind, payload_json, reason, actor_user_id
		 FROM operator_case_actions WHERE case_id = ? ORDER BY id DESC LIMIT ?`,
		caseID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []OperatorCaseActionRow
	for rows.Next() {
		var r OperatorCaseActionRow
		var payload []byte
		if err := rows.Scan(&r.ID, &r.CaseID, &r.CreatedAt, &r.ActionKind, &payload, &r.Reason, &r.ActorUserID); err != nil {
			return nil, err
		}
		if len(payload) > 0 {
			r.PayloadJSON = append([]byte(nil), payload...)
		}
		out = append(out, r)
	}
	if out == nil {
		out = []OperatorCaseActionRow{}
	}
	return out, rows.Err()
}
