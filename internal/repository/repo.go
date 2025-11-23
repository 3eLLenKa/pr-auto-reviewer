package repository

import (
	"database/sql"

	pg_pr "github.com/3eLLenKa/test-avito/internal/repository/postgres/pr"
	pg_team "github.com/3eLLenKa/test-avito/internal/repository/postgres/team"
	pg_user "github.com/3eLLenKa/test-avito/internal/repository/postgres/user"
)

type Repositories struct {
	PullRequest *pg_pr.PRRepo
	Team        *pg_team.TeamRepo
	User        *pg_user.UserRepo
}

func New(db *sql.DB) *Repositories {
	return &Repositories{
		PullRequest: pg_pr.New(db),
		Team:        pg_team.New(db),
		User:        pg_user.New(db),
	}
}
