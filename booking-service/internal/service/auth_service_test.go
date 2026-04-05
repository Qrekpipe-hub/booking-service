package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/example/booking-service/internal/model"
	"github.com/example/booking-service/internal/service"
)

type mockUserRepo struct {
	users map[string]*model.User
}

func newMockUserRepo() *mockUserRepo { return &mockUserRepo{users: map[string]*model.User{}} }
func (r *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}
func (r *mockUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	return r.users[email], nil
}
func (r *mockUserRepo) Create(_ context.Context, u *model.User) error {
	r.users[u.Email] = u
	return nil
}

func TestDummyLogin_AdminToken(t *testing.T) {
	svc := service.NewAuthService(newMockUserRepo(), "test-secret")

	token, err := svc.DummyLogin(model.RoleAdmin)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := svc.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, service.DummyAdminID.String(), claims.UserID)
	assert.Equal(t, string(model.RoleAdmin), claims.Role)
}

func TestDummyLogin_UserToken(t *testing.T) {
	svc := service.NewAuthService(newMockUserRepo(), "test-secret")

	token, err := svc.DummyLogin(model.RoleUser)
	require.NoError(t, err)

	claims, err := svc.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, service.DummyUserID.String(), claims.UserID)
	assert.Equal(t, string(model.RoleUser), claims.Role)
}

func TestDummyLogin_InvalidRole(t *testing.T) {
	svc := service.NewAuthService(newMockUserRepo(), "test-secret")
	_, err := svc.DummyLogin("superuser")
	assert.Error(t, err)
}

func TestParseToken_InvalidSignature(t *testing.T) {
	svc := service.NewAuthService(newMockUserRepo(), "secret-A")

	token, _ := svc.DummyLogin(model.RoleAdmin)

	svc2 := service.NewAuthService(newMockUserRepo(), "secret-B")
	_, err := svc2.ParseToken(token)
	assert.Error(t, err, "token signed with different secret must fail")
}

func TestRegisterAndLogin(t *testing.T) {
	repo := newMockUserRepo()
	svc := service.NewAuthService(repo, "secret")

	user, err := svc.Register(context.Background(), "alice@example.com", "password123", model.RoleUser)
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", user.Email)

	token, err := svc.Login(context.Background(), "alice@example.com", "password123")
	require.NoError(t, err)

	claims, err := svc.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, user.ID.String(), claims.UserID)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	svc := service.NewAuthService(repo, "secret")

	_, err := svc.Register(context.Background(), "bob@example.com", "pass", model.RoleUser)
	require.NoError(t, err)

	_, err = svc.Register(context.Background(), "bob@example.com", "pass2", model.RoleUser)
	assert.ErrorIs(t, err, service.ErrEmailTaken)
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	svc := service.NewAuthService(repo, "secret")

	_, _ = svc.Register(context.Background(), "carol@example.com", "correct", model.RoleUser)

	_, err := svc.Login(context.Background(), "carol@example.com", "wrong")
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}
