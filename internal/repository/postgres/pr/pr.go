package pg_pr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/3eLLenKa/test-avito/internal/domain"
)

type PRRepo struct {
	db *sql.DB
}

func New(db *sql.DB) *PRRepo {
	return &PRRepo{db: db}
}

// вспомогательная функция для одновременного получения
// основных полей PR и всех назначенных ревьюверов для группы PR
func (r *PRRepo) scanPRsWithReviewers(ctx context.Context, rowsPRs *sql.Rows) ([]*domain.PullRequest, error) {

	prsMap := make(map[string]*domain.PullRequest)
	prIDs := make([]string, 0)

	for rowsPRs.Next() {
		pr := &domain.PullRequest{AssignedReviewers: make([]string, 0)}
		var mergedAt sql.NullTime

		if err := rowsPRs.Scan(
			&pr.PullRequestId,
			&pr.PullRequestName,
			&pr.AuthorId,
			&pr.Status,
			&pr.CreatedAt,
			&mergedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning pull request row: %w", err)
		}
		if mergedAt.Valid {
			pr.MergedAt = &mergedAt.Time
		}

		prsMap[pr.PullRequestId] = pr
		prIDs = append(prIDs, pr.PullRequestId)
	}
	if rowsPRs.Err() != nil {
		return nil, fmt.Errorf("rows iteration error after scanning PRs: %w", rowsPRs.Err())
	}

	if len(prIDs) == 0 {
		return []*domain.PullRequest{}, nil
	}

	queryReviewers := `
        SELECT pull_request_id, reviewer_id
        FROM pull_request_reviewers
        WHERE pull_request_id = ANY($1)
    `

	rowsReviewers, err := r.db.QueryContext(ctx, queryReviewers, prIDs)
	if err != nil {
		return nil, fmt.Errorf("error executing reviewers query: %w", err)
	}
	defer rowsReviewers.Close()

	for rowsReviewers.Next() {
		var prID, reviewerID string
		if err := rowsReviewers.Scan(&prID, &reviewerID); err != nil {
			return nil, fmt.Errorf("error scanning reviewer row: %w", err)
		}
		if pr, ok := prsMap[prID]; ok {
			pr.AssignedReviewers = append(pr.AssignedReviewers, reviewerID)
		}
	}
	if rowsReviewers.Err() != nil {
		return nil, fmt.Errorf("rows iteration error after scanning reviewers: %w", rowsReviewers.Err())
	}

	prs := make([]*domain.PullRequest, 0, len(prsMap))
	for _, pr := range prsMap {
		prs = append(prs, pr)
	}
	return prs, nil
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
		return nil, fmt.Errorf("failed to scan PR %s data: %w", prID, err)
	}
	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}

	rows, err := r.db.Query("SELECT reviewer_id FROM pull_request_reviewers WHERE pull_request_id = $1", prID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reviewers for PR %s: %w", prID, err)
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, fmt.Errorf("failed to scan reviewer ID for PR %s: %w", prID, err)
		}
		reviewers = append(reviewers, reviewerID)
	}
	pr.AssignedReviewers = reviewers

	if rows.Err() != nil {
		return nil, fmt.Errorf("rows iteration error while scanning reviewers for PR %s: %w", prID, rows.Err())
	}

	return pr, nil
}

func (r *PRRepo) Create(ctx context.Context, prId, prName, authorId string, reviewers []string, createdAt time.Time) (*domain.PullRequest, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction for PR create: %w", err)
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
			return nil, fmt.Errorf("failed to insert reviewer %s for PR %s: %w", reviewerID, prId, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction for PR create: %w", err)
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
	res, err := r.db.ExecContext(ctx, query, pr.Status, pr.MergedAt, pr.PullRequestId)
	if err != nil {
		return nil, fmt.Errorf("failed to execute update PR %s query: %w", pr.PullRequestId, err)
	}

	if rows, _ := res.RowsAffected(); rows == 0 {
		return nil, domain.ErrPRNotFound
	}

	return pr, nil
}

func (r *PRRepo) Reassign(ctx context.Context, prId, oldUserId, newUserId string) (*domain.PullRequest, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction for PR reassign: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, "DELETE FROM pull_request_reviewers WHERE pull_request_id = $1 AND reviewer_id = $2", prId, oldUserId)
	if err != nil {
		return nil, fmt.Errorf("failed to delete old reviewer %s: %w", oldUserId, err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return nil, domain.ErrNotAssigned
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id) VALUES ($1, $2)", prId, newUserId)
	if err != nil {
		return nil, fmt.Errorf("failed to insert new reviewer %s: %w", newUserId, err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction for PR reassign: %w", err)
	}

	return r.GetPR(ctx, prId)
}

func (r *PRRepo) ListPRs(ctx context.Context) ([]*domain.PullRequest, error) {
	query := `
        SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
        FROM pull_requests
    `
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all pull requests: %w", err)
	}
	defer rows.Close()

	return r.scanPRsWithReviewers(ctx, rows)
}

func (r *PRRepo) ListOpenPRsByReviewers(ctx context.Context, deactivatedUserIDs []string) ([]*domain.PullRequest, error) {
	queryPRs := `
        SELECT 
            pr.pull_request_id, 
            pr.pull_request_name, 
            pr.author_id, 
            pr.status, 
            pr.created_at, 
            pr.merged_at
        FROM pull_requests pr
        WHERE pr.status = $2 -- Предполагается, что 'OPEN' это domain.PRStatusOpen
          AND EXISTS (
              SELECT 1 FROM pull_request_reviewers prr
              WHERE prr.pull_request_id = pr.pull_request_id
                AND prr.reviewer_id = ANY($1) 
          )
    `
	rows, err := r.db.QueryContext(ctx, queryPRs, deactivatedUserIDs, domain.PRStatusOpen)
	if err != nil {
		return nil, fmt.Errorf("error executing ListOpenPRsByReviewers query: %w", err)
	}
	defer rows.Close()

	return r.scanPRsWithReviewers(ctx, rows)
}
