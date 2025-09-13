package auth

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	rpcstatus "google.golang.org/grpc/status"
)

// ListMyRoles returns a list of the user's roles.
func (s *Auth) ListMyRoles(ctx context.Context) ([]*Role, error) {
	claims := ClaimsFromContext(ctx)

	zlog := s.zlog.With(
		zap.String("Method", "ListMyRoles"),
		zap.String("username", claims.Username),
	)

	roles, err := listRolesByUsername(ctx, s.db, claims.Username)
	if err != nil {
		zlog.Error("failed to list roles", zap.Error(err))
		return nil, err
	}

	return roles, nil
}

// CanI checks if the user has the specified permissions.
func (s *Auth) CanI(ctx context.Context, permissions ...string) error {
	if len(permissions) == 0 {
		return errors.New("no permissions specified")
	}

	claims := ClaimsFromContext(ctx)
	roles, err := listRolesByUsername(ctx, s.db, claims.Username)
	if err != nil {
		return err
	}

	missed := checkMissingPermissions(permissions, roleToPermissions(roles))
	if len(missed) > 0 {
		return rpcstatus.Error(
			codes.PermissionDenied,
			fmt.Sprintf("You do not have sufficient permissions. Required %v to perform this action", missed),
		)
	}

	return nil
}

type Role struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Permissions []string `json:"permissions,omitempty"`
}

type Permission struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// roleToPermissions converts a list of roles to a list of permissions.
func roleToPermissions(rs []*Role) []string {
	ps := make([]string, 0)
	for _, r := range rs {
		ps = append(ps, r.Permissions...)
	}

	return ps
}

// checkMissingPermissions returns a list of permissions that are in the
// 'wanted' slice but are not in the 'having' slice. It's useful for
// determining what permissions a user is missing for a given action.
func checkMissingPermissions(wanted, having []string) []string {
	havingSet := make(map[string]struct{}, len(having))
	for _, p := range having {
		havingSet[p] = struct{}{}
	}

	missing := make([]string, 0)
	for _, p := range wanted {
		if _, exists := havingSet[p]; !exists {
			missing = append(missing, p)
		}
	}

	return missing
}
