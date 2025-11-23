package domain

import "time"

type User struct {
	ID       string
	Name     string
	IsActive bool
	TeamName string
}

type Team struct {
	Name    string
	Members []User
}

type PullRequestStatus string

type PullRequest struct {
	PullRequestId     string
	PullRequestName   string
	AuthorId          string
	AssignedReviewers []string
	Status            PullRequestStatus
	CreatedAt         *time.Time
	MergedAt          *time.Time
}

type AssignmentCountByUser struct {
	UserID           string
	AssignmentsCount int
}

type AssignmentCountByPR struct {
	PullRequestID  string
	ReviewersCount int
}
