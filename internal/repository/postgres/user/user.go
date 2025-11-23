package pg_user

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/3eLLenKa/test-avito/internal/domain"
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

func (r *UserRepo) DeactivateByTeam(ctx context.Context, teamName string) ([]string, error) {
	query := `
        UPDATE users 
        SET is_active = FALSE 
        WHERE team_name = $1 AND is_active = TRUE
        RETURNING user_id;
    `

	rows, err := r.db.QueryContext(ctx, query, teamName)
	if err != nil {
		return nil, fmt.Errorf("error executing bulk deactivation query for team %s: %w", teamName, err)
	}
	defer rows.Close()

	deactivatedIDs := make([]string, 0)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("error scanning deactivated user ID for team %s: %w", teamName, err)
		}
		deactivatedIDs = append(deactivatedIDs, userID)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("rows iteration error after scanning: %w", rows.Err())
	}

	return deactivatedIDs, nil
}

func (r *UserRepo) ListActiveMembersByTeam(ctx context.Context, teamName string, excludeUserID string) ([]domain.User, error) {
	query := `
        SELECT user_id, username, team_name, is_active
        FROM users
        WHERE team_name = $1 
          AND is_active = TRUE 
          AND user_id != $2;
    `
	rows, err := r.db.QueryContext(ctx, query, teamName, excludeUserID)
	if err != nil {
		return nil, fmt.Errorf("error executing ListActiveMembersByTeam query for team %s: %w", teamName, err)
	}
	defer rows.Close()

	members := make([]domain.User, 0)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Name, &u.TeamName, &u.IsActive); err != nil {
			return nil, fmt.Errorf("error scanning active user row for team %s: %w", teamName, err)
		}
		members = append(members, u)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("rows iteration error in ListActiveMembersByTeam: %w", rows.Err())
	}

	return members, nil
}
