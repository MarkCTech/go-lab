package authstore

import (
	"context"
	"errors"
)

// ListPlatformRoleNamesForUser returns distinct role names assigned in user_platform_roles.
func (s *Store) ListPlatformRoleNamesForUser(ctx context.Context, userID int) ([]string, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("nil db")
	}
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT r.name
		 FROM platform_roles r
		 INNER JOIN user_platform_roles ur ON ur.role_id = r.id
		 WHERE ur.user_id = ?
		 ORDER BY r.name ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}
