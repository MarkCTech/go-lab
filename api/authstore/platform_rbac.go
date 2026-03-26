package authstore

import (
	"context"
	"errors"

	"github.com/go-sql-driver/mysql"
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
		 INNER JOIN operator_account_roles oar ON oar.role_id = r.id
		 INNER JOIN operator_accounts oa ON oa.id = oar.operator_account_id
		 INNER JOIN users u ON u.id = oa.linked_user_id
		 WHERE oa.linked_user_id = ? AND oa.status = 'active' AND u.deleted_at IS NULL
		 ORDER BY r.name ASC`,
		userID,
	)
	if err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1146 {
			rows, err = s.db.QueryContext(ctx,
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
		} else {
			return nil, err
		}
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
