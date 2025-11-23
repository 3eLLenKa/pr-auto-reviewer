package pg_team

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/3eLLenKa/test-avito/internal/domain"
)

var (
	ErrTeamExists   = fmt.Errorf("TEAM_EXISTS: team_name already exists")
	ErrTeamNotFound = fmt.Errorf("NOT_FOUND: team not found")
)

type TeamRepo struct {
	db *sql.DB
}

func New(db *sql.DB) *TeamRepo {
	return &TeamRepo{db: db}
}

func (r *TeamRepo) Add(ctx context.Context, teamName string, members []domain.User) (*domain.Team, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "INSERT INTO teams (team_name) VALUES ($1) ON CONFLICT (team_name) DO NOTHING", teamName)
	if err != nil {
		return nil, err
	}

	for _, member := range members {
		query := `
			INSERT INTO users (user_id, username, team_name, is_active)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) DO UPDATE SET
				username = EXCLUDED.username,
				team_name = EXCLUDED.team_name,
				is_active = EXCLUDED.is_active
		`
		_, err = tx.ExecContext(ctx, query, member.ID, member.Name, teamName, member.IsActive)
		if err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetTeam(ctx, teamName)
}

func (r *TeamRepo) GetTeam(ctx context.Context, teamName string) (*domain.Team, error) {
	var teamID string
	err := r.db.QueryRowContext(ctx, "SELECT team_name FROM teams WHERE team_name = $1", teamName).Scan(&teamID)
	if err == sql.ErrNoRows {
		return nil, domain.ErrTeamNotFound
	}
	if err != nil {
		return nil, err
	}

	query := "SELECT user_id, username, is_active FROM users WHERE team_name = $1"
	rows, err := r.db.QueryContext(ctx, query, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []domain.User
	for rows.Next() {
		var member domain.User
		member.TeamName = teamName

		if err := rows.Scan(&member.ID, &member.Name, &member.IsActive); err != nil {
			return nil, err
		}

		members = append(members, member)
	}

	return &domain.Team{
		Name:    teamName,
		Members: members,
	}, rows.Err()
}
