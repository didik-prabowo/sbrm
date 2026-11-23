package service

import (
	"context"
	"errors"
	"strings"

	ierr "github.com/didik-prabowo/sbrm/internal/domain/error"
	"github.com/didik-prabowo/sbrm/internal/repository/postgres"
	"github.com/didik-prabowo/sbrm/pkg/bcrypt"
	"github.com/didik-prabowo/sbrm/pkg/jwt"
)

type AuthService interface {
	Login(ctx context.Context, email, password string) (string, error)
}

type authServiceImpl struct {
	hasher     bcrypt.BcryptHasher
	jwtManager jwt.Manager
	UserRepo   postgres.UserRepository
}

func NewAuthService(hasher bcrypt.BcryptHasher, jwtManager jwt.Manager, userRepo postgres.UserRepository) AuthService {
	return &authServiceImpl{
		hasher:     hasher,
		UserRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

func (s *authServiceImpl) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return "", ierr.ValidationError("email dan password wajib diisi")
	}
	user, err := s.UserRepo.FindByEmail(ctx, email)
	if errors.Is(err, ierr.ErrRecordNotFound) {
		return "", ierr.ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}

	if !s.hasher.Compare(user.PasswordHash, password) {
		return "", ierr.ErrInvalidCredentials
	}

	return s.jwtManager.Sign(user.ID, string(user.Role))
}
