package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	edpb "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	rpcstatus "google.golang.org/grpc/status"
)

type Auth struct {
	db   *pgxpool.Pool
	aKey paseto.V4SymmetricKey
	rKey paseto.V4SymmetricKey
	zlog *zap.Logger
}

func New(_ context.Context, db *pgxpool.Pool, zlog *zap.Logger, aKey, rKey paseto.V4SymmetricKey) (*Auth, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	if zlog == nil {
		return nil, errors.New("logger is nil")
	}

	return &Auth{
		db:   db,
		aKey: aKey,
		rKey: rKey,
		zlog: zlog,
	}, nil
}

func (s *Auth) Profile(ctx context.Context) (*User, error) {
	claims := ClaimsFromContext(ctx)

	s.zlog.With(
		zap.String("Method", "Profile"),
		zap.String("Username", claims.Username),
	)

	user, err := getUserByUsername(ctx, s.db, claims.Username)
	if errors.Is(err, ErrUserNotFound) {
		return nil, rpcstatus.Error(codes.PermissionDenied, "You are not allowed to access this resource or (it may not exist)")
	}
	if err != nil {
		s.zlog.Error("failed to get user", zap.Error(err))
		return nil, err
	}

	return user, nil
}

func (s *Auth) Login(ctx context.Context, in *LoginReq) (*Token, error) {
	zlog := s.zlog.With(
		zap.String("Method", "Login"),
		zap.String("username", in.Username),
	)

	if err := in.Validate(); err != nil {
		return nil, err
	}

	user, err := getUserByUsername(ctx, s.db, in.Username)
	if errors.Is(err, ErrUserNotFound) {
		return nil, rpcstatus.Error(codes.Unauthenticated, "Your credentials not valid. Please check your username and password and try again.")
	}
	if err != nil {
		zlog.Error("failed to get user", zap.Error(err))
		return nil, err
	}

	if passed := user.ComparePassword(in.Password); !passed {
		return nil, rpcstatus.Error(codes.Unauthenticated, "Your credentials not valid. Please check your username and password and try again.")
	}
	if !user.IsEnabled() {
		return nil, rpcstatus.Error(codes.Unauthenticated, "Your credentials not valid. Please check your token and try again.")
	}

	token, err := s.genToken(user)
	if err != nil {
		zlog.Error("failed to generate token", zap.Error(err))
		return nil, err
	}

	return token, nil
}

type NewTokenReq struct {
	Token string `json:"token"`
}

func (s *Auth) RefreshToken(ctx context.Context, in *NewTokenReq) (*Token, error) {
	zlog := s.zlog.With(
		zap.String("Method", "RefreshToken"),
		zap.Any("req", in),
	)

	if in.Token == "" {
		return nil, rpcstatus.Error(codes.Unauthenticated, "Your credentials not valid. Please check your token and try again.")
	}

	rules := []paseto.Rule{
		paseto.NotExpired(),
		paseto.ValidAt(time.Now()),
	}

	parser := paseto.MakeParser(rules)
	t, err := parser.ParseV4Local(s.rKey, in.Token, nil)
	if err != nil {
		return nil, rpcstatus.Error(codes.Unauthenticated, "Your credentials not valid. Please check your token and try again.")
	}

	claims := new(Claims)
	if err := t.Get("profile", claims); err != nil {
		return nil, rpcstatus.Error(codes.Unauthenticated, "Your credentials not valid. Please check your token and try again.")
	}

	user, err := getUserByUsername(ctx, s.db, claims.Username)
	if errors.Is(err, ErrUserNotFound) {
		return nil, rpcstatus.Error(codes.Unauthenticated, "Your credentials not valid. Please check your token and try again.")
	}
	if err != nil {
		zlog.Error("failed to get user", zap.Error(err))
		return nil, err
	}
	if !user.IsEnabled() {
		return nil, rpcstatus.Error(codes.Unauthenticated, "Your credentials not valid. Please check your token and try again.")
	}

	token, err := s.genToken(user)
	if err != nil {
		zlog.Error("failed to generate token", zap.Error(err))
		return nil, err
	}

	return token, nil
}

func (s *Auth) genToken(u *User) (*Token, error) {
	now := time.Now()

	t := paseto.NewToken()
	t.SetSubject(u.Username)
	t.SetIssuedAt(now)
	t.SetNotBefore(now)
	t.SetExpiration(now.Add(time.Hour))
	t.SetFooter([]byte(now.Format(time.RFC3339)))

	if err := t.Set("profile", u.toClaims()); err != nil {
		return nil, fmt.Errorf("failed to set claims: %w", err)
	}

	accessToken := t.V4Encrypt(s.aKey, nil)

	t.SetExpiration(now.Add(time.Hour * 24 * 7))
	refreshToken := t.V4Encrypt(s.rKey, nil)

	return &Token{
		Access:  accessToken,
		Refresh: refreshToken,
	}, nil
}

type Token struct {
	Access  string `json:"accessToken"`
	Refresh string `json:"refreshToken"`
}

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r *LoginReq) Validate() error {
	violations := make([]*edpb.BadRequest_FieldViolation, 0)

	if r.Username == "" {
		violations = append(violations, &edpb.BadRequest_FieldViolation{
			Field:       "username",
			Description: "Username must not be empty",
		})
	}

	if r.Password == "" {
		violations = append(violations, &edpb.BadRequest_FieldViolation{
			Field:       "password",
			Description: "Password must not be empty",
		})
	}

	if len(violations) > 0 {
		s, _ := rpcstatus.New(
			codes.InvalidArgument,
			"Credentials are not valid or incomplete. Please check the errors and try again, see details for more information.",
		).WithDetails(&edpb.BadRequest{
			FieldViolations: violations,
		})

		return s.Err()
	}

	return nil
}

type User struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	Status      status    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`

	hashedPassword []byte
	createdBy      string
	updatedBy      string
	updatedAt      time.Time
}

func (u *User) IsEnabled() bool {
	return u.Status == StatusEnabled
}

func (u *User) toClaims() *Claims {
	return &Claims{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
	}
}

func (u *User) ComparePassword(password string) bool {
	return bcrypt.CompareHashAndPassword(u.hashedPassword, []byte(password)) == nil
}
