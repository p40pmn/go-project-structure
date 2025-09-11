package auth

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
)

// ErrUserNotFound is returned when a user is not found.
var ErrUserNotFound = errors.New("user not found")

func getUserByUsername(ctx context.Context, db *pgxpool.Pool, username string) (*User, error) {
	q, args := sq.Select(
		"id",
		"username",
		"display_name",
		"hash_password",
		"status",
		"created_at",
		"created_by",
		"updated_at",
		"updated_by",
	).
		Where(
			sq.Eq{
				"username": username,
			}).
		PlaceholderFormat(sq.Dollar).
		Limit(1).
		MustSql()

	row := db.QueryRow(ctx, q, args...)
	u := new(User)
	err := row.Scan(
		&u.ID,
		&u.Username,
		&u.DisplayName,
		&u.hashedPassword,
		&u.Status,
		&u.CreatedAt,
		&u.createdBy,
		&u.updatedAt,
		&u.updatedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return u, nil
}

func listRolesByUsername(ctx context.Context, db *pgxpool.Pool, username string) ([]*Role, error) {
	query, args := sq.Select(
		"r.id",
		"r.name",
		"r.display_name",
		"ARRAY_AGG(rh.permission_name)",
	).
		From("user_has_role AS ur").
		InnerJoin("role r ON r.name = ur.role_name").
		InnerJoin("role_has_permission AS rh ON rh.role_name = ur.role_name").
		Where(
			sq.Eq{
				"ur.username": username,
			},
		).
		GroupBy(
			"r.id",
			"r.name",
			"r.display_name",
			"ur.username",
		).
		PlaceholderFormat(sq.Dollar).
		MustSql()

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	rs := make([]*Role, 0)
	for rows.Next() {
		r := new(Role)
		if err := rows.Scan(
			&r.ID,
			&r.Name,
			&r.DisplayName,
			pq.Array(&r.Permissions),
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rs = append(rs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate rows: %w", err)
	}

	return rs, nil
}
