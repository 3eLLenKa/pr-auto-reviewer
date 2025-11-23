package pg_pr

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/3eLLenKa/test-avito/internal/domain"
)

type PRRepo struct {
	db *sql.DB
}

func New(db *sql.DB) *PRRepo {
	return &PRRepo{db: db}
}

func (r *PRRepo) toDomainPR(row *sql.Row, prID string) (*domain.PullRequest, error) {
	pr := &domain.PullRequest{PullRequestId: prID}
	var mergedAt sql.NullTime

	err := row.Scan(
		&pr.PullRequestName,
		&pr.AuthorId,
		&pr.Status,
		&pr.CreatedAt,
		&mergedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrPRNotFound
	}
	if err != nil {
		return nil, err
	}
	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}

	rows, err := r.db.Query("SELECT reviewer_id FROM pull_request_reviewers WHERE pull_request_id = $1", prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, reviewerID)
	}
	pr.AssignedReviewers = reviewers

	return pr, rows.Err()
}

func (r *PRRepo) Create(ctx context.Context, prId, prName, authorId string, reviewers []string, createdAt time.Time) (*domain.PullRequest, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = tx.ExecContext(ctx, query, prId, prName, authorId, domain.PRStatusOpen, createdAt)
	if err != nil {
		return nil, domain.ErrPRExists
	}

	for _, reviewerID := range reviewers {
		_, err = tx.ExecContext(ctx, "INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id) VALUES ($1, $2)", prId, reviewerID)
		if err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &domain.PullRequest{
		PullRequestId:     prId,
		PullRequestName:   prName,
		AuthorId:          authorId,
		Status:            domain.PRStatusOpen,
		AssignedReviewers: reviewers,
		CreatedAt:         &createdAt,
	}, nil
}

func (r *PRRepo) GetPR(ctx context.Context, prId string) (*domain.PullRequest, error) {
	query := `
		SELECT pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE pull_request_id = $1
	`
	return r.toDomainPR(r.db.QueryRowContext(ctx, query, prId), prId)
}

func (r *PRRepo) UpdatePR(ctx context.Context, pr *domain.PullRequest) (*domain.PullRequest, error) {
	if pr.Status != domain.PRStatusMerged {
		return nil, errors.New("updatePR is only for merge status in this implementation")
	}

	pr.MergedAt = new(time.Time)
	*pr.MergedAt = time.Now().In(time.UTC)

	query := `
		UPDATE pull_requests
		SET status = $1, merged_at = $2
		WHERE pull_request_id = $3
	`
	_, err := r.db.ExecContext(ctx, query, pr.Status, pr.MergedAt, pr.PullRequestId)
	if err != nil {
		return nil, err
	}

	return pr, nil
}

func (r *PRRepo) Reassign(ctx context.Context, prId, oldUserId, newUserId string) (*domain.PullRequest, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, "DELETE FROM pull_request_reviewers WHERE pull_request_id = $1 AND reviewer_id = $2", prId, oldUserId)
	if err != nil {
		return nil, err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return nil, domain.ErrNotAssigned
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id) VALUES ($1, $2)", prId, newUserId)

	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetPR(ctx, prId)
}

func (r *PRRepo) ListPRs(ctx context.Context) ([]*domain.PullRequest, error) {
	query := "SELECT pull_request_id FROM pull_requests"
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		prIDs = append(prIDs, id)
	}

	var prs []*domain.PullRequest
	for _, id := range prIDs {
		pr, err := r.GetPR(ctx, id)
		if err != nil {
			return nil, err
		}

		prs = append(prs, pr)
	}

	return prs, rows.Err()
}
