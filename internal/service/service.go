package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/3eLLenKa/test-avito/internal/domain"
)

type PullRequestRepo interface {
	Create(ctx context.Context, prId, prName, authorId string, reviewers []string, createdAt time.Time) (*domain.PullRequest, error)
	UpdatePR(ctx context.Context, pr *domain.PullRequest) (*domain.PullRequest, error)
	GetPR(ctx context.Context, prId string) (*domain.PullRequest, error)
	Reassign(ctx context.Context, prId, oldUserId, newUserId string) (*domain.PullRequest, error)
	ListPRs(ctx context.Context) ([]*domain.PullRequest, error)
}

type TeamRepo interface {
	Add(ctx context.Context, teamName string, members []domain.User) (*domain.Team, error)
	GetTeam(ctx context.Context, teamName string) (*domain.Team, error)
}

type UserRepo interface {
	SetUserActive(ctx context.Context, userId string, isActive bool) (*domain.User, error)
	GetUserById(ctx context.Context, userId string) (*domain.User, error)
}

type Service struct {
	pr   PullRequestRepo
	team TeamRepo
	user UserRepo
}

func New(pr PullRequestRepo, team TeamRepo, user UserRepo) *Service {
	return &Service{
		pr:   pr,
		team: team,
		user: user,
	}
}

func (s *Service) PullRequestCreate(ctx context.Context, prId, prName, authorId string) (*domain.PullRequest, error) {
	author, err := s.user.GetUserById(ctx, authorId)
	if err != nil {
		return nil, err
	}

	authorTeam, err := s.team.GetTeam(ctx, author.TeamName)
	if err != nil {
		return nil, err
	}

	if len(authorTeam.Members) == 0 {
		return nil, domain.ErrNoCandidate
	}

	candidates := []domain.User{}
	for _, m := range authorTeam.Members {
		if m.ID != authorId && m.IsActive {
			candidates = append(candidates, m)
		}
	}

	reviewers := reviewersIds(pickRandomUsers(candidates, 2))
	createdAt := time.Now()

	return s.pr.Create(ctx, prId, prName, authorId, reviewers, createdAt)
}

func (s *Service) PullRequestMerge(ctx context.Context, prId string) (*domain.PullRequest, error) {
	pr, err := s.pr.GetPR(ctx, prId)
	if err != nil {
		return nil, err
	}

	if pr.Status == domain.PRStatusMerged {
		return pr, nil
	}

	pr.Status = domain.PRStatusMerged

	return s.pr.UpdatePR(ctx, pr)
}

func (s *Service) PullRequestReassign(ctx context.Context, prId, oldUserId string) (*domain.PullRequest, string, error) {
	pr, err := s.pr.GetPR(ctx, prId)
	if err != nil {
		return nil, "", err
	}

	if pr.Status == domain.PRStatusMerged {
		return nil, "", domain.ErrPRMerged
	}

	found := false
	for _, uid := range pr.AssignedReviewers {
		if uid == oldUserId {
			found = true
			break
		}
	}
	if !found {
		return nil, "", domain.ErrNotAssigned
	}

	oldUser, err := s.user.GetUserById(ctx, oldUserId)
	if err != nil {
		return nil, "", err
	}

	oldUserTeam, err := s.team.GetTeam(ctx, oldUser.TeamName)
	if err != nil {
		return nil, "", err
	}

	candidates := make([]domain.User, 0)
	assignedMap := make(map[string]bool, len(pr.AssignedReviewers))
	for _, uid := range pr.AssignedReviewers {
		assignedMap[uid] = true
	}

	for _, m := range oldUserTeam.Members {
		if m.ID == oldUserId || m.ID == pr.AuthorId || !m.IsActive || assignedMap[m.ID] {
			continue
		}
		candidates = append(candidates, m)
	}

	if len(candidates) == 0 {
		return nil, "", domain.ErrNoCandidate
	}

	rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
	newReviewer := candidates[0]

	updatedPR, err := s.pr.Reassign(ctx, prId, oldUserId, newReviewer.ID)
	if err != nil {
		return nil, "", err
	}

	return updatedPR, newReviewer.ID, nil
}

func (s *Service) TeamAdd(ctx context.Context, teamName string, members []domain.User) (*domain.Team, error) {
	return s.team.Add(ctx, teamName, members)
}

func (s *Service) TeamGet(ctx context.Context, teamName string) (*domain.Team, error) {
	return s.team.GetTeam(ctx, teamName)
}

func (s *Service) UsersGetReview(ctx context.Context, userId string) ([]*domain.PullRequest, error) {
	PRs, err := s.pr.ListPRs(ctx)
	if err != nil {
		return nil, err
	}

	res := []*domain.PullRequest{}
	for _, pr := range PRs {
		for _, ar := range pr.AssignedReviewers {
			if ar == userId {
				res = append(res, pr)
			}
		}
	}

	return res, nil
}

func (s *Service) SetUserActive(ctx context.Context, userId string, isActive bool) (*domain.User, error) {
	return s.user.SetUserActive(ctx, userId, isActive)
}

func pickRandomUsers(users []domain.User, count int) []domain.User {
	if len(users) == 0 {
		return nil
	}

	rand.Shuffle(len(users), func(i, j int) { users[i], users[j] = users[j], users[i] })

	if len(users) < count {
		return users
	}
	return users[:count]
}

func reviewersIds(users []domain.User) []string {
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.ID
	}
	return ids
}
