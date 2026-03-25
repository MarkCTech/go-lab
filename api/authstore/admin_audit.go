package authstore

import (
	"context"
	"errors"
)

// InsertAdminAuditEvent appends an immutable control-plane audit row (separate from auth_audit_events).
func (s *Store) InsertAdminAuditEvent(ctx context.Context, actorUserID *int, authSubject, action, resourceType, resourceID, reason, requestID, ip, ua string, metaJSON []byte) error {
	if s == nil || s.db == nil {
		return errors.New("nil db")
	}
	var uid any
	if actorUserID != nil {
		uid = *actorUserID
	} else {
		uid = nil
	}
	var meta any
	if len(metaJSON) > 0 {
		meta = metaJSON
	} else {
		meta = nil
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO admin_audit_events
		 (actor_user_id, auth_subject, action, resource_type, resource_id, reason, request_id, ip, user_agent, meta_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uid, authSubject, action, resourceType, nullStr(resourceID),
		nullStr(reason), nullStr(requestID), nullStr(ip), nullStr(ua), meta,
	)
	return err
}
