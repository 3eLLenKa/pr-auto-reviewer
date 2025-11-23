package domain

import "errors"

var (
	ErrPRExists    = errors.New("PR_EXISTS: PR id already exists")
	ErrPRNotFound  = errors.New("NOT_FOUND: pull request not found")
	ErrPRMerged    = errors.New("PR_MERGED: cannot reassign on merged PR")
	ErrNotAssigned = errors.New("NOT_ASSIGNED: reviewer is not assigned to this PR")
	ErrNoCandidate = errors.New("NO_CANDIDATE: no active replacement candidate in team")

	ErrTeamExists   = errors.New("TEAM_EXISTS: team_name already exists")
	ErrTeamNotFound = errors.New("NOT_FOUND: team not found")

	ErrUserNotFound  = errors.New("user not found")
	ErrNoAssignedPRs = errors.New("no PRs assigned to this user")
)
