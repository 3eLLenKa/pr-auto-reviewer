package pg_user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/3eLLenKa/test-avito/internal/domain"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrNoAssignedPRs = errors.New("no PRs assigned to this user")
)

type UserRepo struct {
	db *sql.DB
}

func New(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetUserById(ctx context.Context, userId string) (*domain.User, error) {
	user := &domain.User{}
	query := "SELECT user_id, username, team_name, is_active FROM users WHERE user_id = $1"
	err := r.db.QueryRowContext(ctx, query, userId).Scan(
		&user.ID,
		&user.Name,
		&user.TeamName,
		&user.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	return user, err
}

func (r *UserRepo) SetUserActive(ctx context.Context, userId string, isActive bool) (*domain.User, error) {
	query := "UPDATE users SET is_active = $1 WHERE user_id = $2 RETURNING username, team_name"
	user := &domain.User{ID: userId, IsActive: isActive}

	err := r.db.QueryRowContext(ctx, query, isActive, userId).Scan(&user.Name, &user.TeamName)
	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}
