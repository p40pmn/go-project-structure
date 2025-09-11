package auth

import "context"

type Claims struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type ctxKey int

const (
	claimsKey ctxKey = iota
)

// ClaimsFromContext retrieves the claims from the context.
// If no claims are found, it returns empty Claims.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	if !ok {
		return &Claims{}
	}

	return claims
}

// ContextWithClaims adds the claims to the context.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}
