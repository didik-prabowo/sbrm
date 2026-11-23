package postgres

import (
	"context"
	"database/sql"
	"errors"

	ierr "github.com/didik-prabowo/sbrm/internal/domain/error"
	user "github.com/didik-prabowo/sbrm/internal/domain/user"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (user.User, error)
}

type userRepositoryImpl struct {
	sqlRepo *SQLRepository
}

func NewUserRepository(sqlRepo *SQLRepository) UserRepository {
	return &userRepositoryImpl{
		sqlRepo: sqlRepo,
	}
}

func (r *userRepositoryImpl) FindByEmail(ctx context.Context, email string) (user.User, error) {
	var (
		u    user.User
		role string
	)

	err := r.sqlRepo.conn(ctx).QueryRowContext(ctx,
		`SELECT id,
					name, email,
					password_hash,
					role
		FROM users
		WHERE LOWER(email) = LOWER($1)`, email,
	).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &role)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.User{}, ierr.ErrRecordNotFound
		}
		return user.User{}, err
	}

	u.Role = user.Role(role)
	return u, nil
}
