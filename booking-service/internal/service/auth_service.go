package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/example/booking-service/internal/model"
	"github.com/example/booking-service/internal/repository"
)

// Fixed UUIDs for dummy users (stable across restarts for test scenarios).
var (
	DummyAdminID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	DummyUserID  = uuid.MustParse("22222222-2222-2222-2222-222222222222")
)

const tokenTTL = 24 * time.Hour

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrEmailTaken = errors.New("email already taken")

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type AuthService struct {
	users     repository.UserRepository
	jwtSecret []byte
}

func NewAuthService(users repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{users: users, jwtSecret: []byte(jwtSecret)}
}

// DummyLogin issues a JWT for a fixed user by role (no DB lookup needed for token).
func (s *AuthService) DummyLogin(role model.Role) (string, error) {
	var userID uuid.UUID
	switch role {
	case model.RoleAdmin:
		userID = DummyAdminID
	case model.RoleUser:
		userID = DummyUserID
	default:
		return "", fmt.Errorf("invalid role: %s", role)
	}
	return s.issueToken(userID, role)
}

// Register creates a new user with a hashed password (optional feature).
func (s *AuthService) Register(ctx context.Context, email, password string, role model.Role) (*model.User, error) {
	existing, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)

	user := &model.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: &hashStr,
		Role:         role,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

// Login verifies credentials and issues a JWT (optional feature).
func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}
	if user == nil || user.PasswordHash == nil {
		return "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}
	return s.issueToken(user.ID, user.Role)
}

// ParseToken validates a JWT and extracts claims.
func (s *AuthService) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

func (s *AuthService) issueToken(userID uuid.UUID, role model.Role) (string, error) {
	claims := &Claims{
		UserID: userID.String(),
		Role:   string(role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
