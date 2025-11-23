package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	ListOpenPRsByReviewers(ctx context.Context, deactivatedUserIDs []string) ([]*domain.PullRequest, error)
}

type TeamRepo interface {
	Add(ctx context.Context, teamName string, members []domain.User) (*domain.Team, error)
	GetTeam(ctx context.Context, teamName string) (*domain.Team, error)
}

type UserRepo interface {
	SetUserActive(ctx context.Context, userId string, isActive bool) (*domain.User, error)
	GetUserById(ctx context.Context, userId string) (*domain.User, error)
	ListActiveMembersByTeam(ctx context.Context, teamName string, excludeUserID string) ([]domain.User, error)
	DeactivateByTeam(ctx context.Context, teamName string) ([]string, error)
}

type Service struct {
	log  *slog.Logger
	pr   PullRequestRepo
	team TeamRepo
	user UserRepo
}

func New(log *slog.Logger, pr PullRequestRepo, team TeamRepo, user UserRepo) *Service {
	return &Service{
		log:  log,
		pr:   pr,
		team: team,
		user: user,
	}
}

func (s *Service) PullRequestCreate(ctx context.Context, prId, prName, authorId string) (*domain.PullRequest, error) {
	author, err := s.user.GetUserById(ctx, authorId)
	if err != nil {
		s.log.Error("service.PullRequestCreate: failed to get author by ID", slog.String("author_id", authorId), slog.Any("error", err))
		return nil, err
	}

	authorTeam, err := s.team.GetTeam(ctx, author.TeamName)
	if err != nil {
		s.log.Error("service.PullRequestCreate: failed to get author team", slog.String("team_name", author.TeamName), slog.Any("error", err))
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

	pr, err := s.pr.Create(ctx, prId, prName, authorId, reviewers, createdAt)
	if err != nil {
		s.log.Error("service.PullRequestCreate: failed to create PR in repo", slog.String("pr_id", prId), slog.Any("error", err))
		return nil, err
	}
	return pr, nil
}

func (s *Service) PullRequestMerge(ctx context.Context, prId string) (*domain.PullRequest, error) {
	pr, err := s.pr.GetPR(ctx, prId)
	if err != nil {
		s.log.Error("service.PullRequestMerge: failed to get PR by ID", slog.String("pr_id", prId), slog.Any("error", err))
		return nil, err
	}

	if pr.Status == domain.PRStatusMerged {
		return pr, nil
	}

	pr.Status = domain.PRStatusMerged

	updatedPR, err := s.pr.UpdatePR(ctx, pr)
	if err != nil {
		s.log.Error("service.PullRequestMerge: failed to update PR status in repo", slog.String("pr_id", prId), slog.Any("error", err))
		return nil, err
	}
	return updatedPR, nil
}

func (s *Service) PullRequestReassign(ctx context.Context, prId, oldUserId string) (*domain.PullRequest, string, error) {
	pr, err := s.pr.GetPR(ctx, prId)
	if err != nil {
		s.log.Error("service.PullRequestReassign: failed to get PR by ID", slog.String("pr_id", prId), slog.Any("error", err))
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
		s.log.Error("service.PullRequestReassign: failed to get old user by ID", slog.String("user_id", oldUserId), slog.Any("error", err))
		return nil, "", err
	}

	oldUserTeam, err := s.team.GetTeam(ctx, oldUser.TeamName)
	if err != nil {
		s.log.Error("service.PullRequestReassign: failed to get team for old user", slog.String("team_name", oldUser.TeamName), slog.Any("error", err))
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
		s.log.Error("service.PullRequestReassign: failed to reassign PR in repo", slog.String("pr_id", prId), slog.String("old_user", oldUserId), slog.String("new_user", newReviewer.ID), slog.Any("error", err))
		return nil, "", err
	}

	return updatedPR, newReviewer.ID, nil
}

func (s *Service) TeamAdd(ctx context.Context, teamName string, members []domain.User) (*domain.Team, error) {
	team, err := s.team.Add(ctx, teamName, members)
	if err != nil {
		s.log.Error("service.TeamAdd: failed to add team in repo", slog.String("team_name", teamName), slog.Any("error", err))
		return nil, err
	}
	return team, nil
}

func (s *Service) TeamGet(ctx context.Context, teamName string) (*domain.Team, error) {
	team, err := s.team.GetTeam(ctx, teamName)
	if err != nil {
		s.log.Error("service.TeamGet: failed to get team from repo", slog.String("team_name", teamName), slog.Any("error", err))
		return nil, err
	}
	return team, nil
}

func (s *Service) UsersGetReview(ctx context.Context, userId string) ([]*domain.PullRequest, error) {
	PRs, err := s.pr.ListPRs(ctx)
	if err != nil {
		s.log.Error("service.UsersGetReview: failed to list all PRs from repo", slog.Any("error", err))
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
	user, err := s.user.SetUserActive(ctx, userId, isActive)
	if err != nil {
		s.log.Error("service.SetUserActive: failed to set user active status in repo", slog.String("user_id", userId), slog.Bool("is_active", isActive), slog.Any("error", err))
		return nil, err
	}
	return user, nil
}

func (s *Service) GetAssignmentStats(ctx context.Context) ([]domain.AssignmentCountByUser, []domain.AssignmentCountByPR, error) {
	prs, err := s.pr.ListPRs(ctx)
	if err != nil {
		s.log.Error("service.GetAssignmentStats: failed to list all PRs from repo", slog.Any("error", err))
		return nil, nil, err
	}

	userCounts := make(map[string]int)
	prCounts := make(map[string]int)

	for _, pr := range prs {
		count := len(pr.AssignedReviewers)
		prCounts[pr.PullRequestId] = count

		for _, reviewerID := range pr.AssignedReviewers {
			userCounts[reviewerID]++
		}
	}

	byUser := make([]domain.AssignmentCountByUser, 0, len(userCounts))
	for id, count := range userCounts {
		byUser = append(byUser, domain.AssignmentCountByUser{UserID: id, AssignmentsCount: count})
	}

	byPR := make([]domain.AssignmentCountByPR, 0, len(prCounts))
	for id, count := range prCounts {
		byPR = append(byPR, domain.AssignmentCountByPR{PullRequestID: id, ReviewersCount: count})
	}

	return byUser, byPR, nil
}

func (s *Service) TeamDeactivateUsers(ctx context.Context, teamName string) ([]string, []*domain.PullRequest, int, int, error) {
	deactivatedUserIDs, err := s.user.DeactivateByTeam(ctx, teamName)
	if err != nil {
		if errors.Is(err, domain.ErrTeamNotFound) {
			return nil, nil, 0, 0, domain.ErrTeamNotFound
		}
		s.log.Error("service.TeamDeactivateUsers: bulk deactivation failed in repo", slog.String("team_name", teamName), slog.Any("error", err))
		return nil, nil, 0, 0, fmt.Errorf("bulk deactivation failed: %w", err)
	}

	if len(deactivatedUserIDs) == 0 {
		return []string{}, nil, 0, 0, nil
	}

	openPRs, err := s.pr.ListOpenPRsByReviewers(ctx, deactivatedUserIDs)
	if err != nil {
		s.log.Error("service.TeamDeactivateUsers: failed to list open PRs by reviewers", slog.Any("deactivated_users", deactivatedUserIDs), slog.Any("error", err))
		return deactivatedUserIDs, nil, 0, 0, fmt.Errorf("failed to list open PRs: %w", err)
	}

	updatedPRs := make([]*domain.PullRequest, 0)
	reassignedCount := 0
	failedCount := 0

	for _, pr := range openPRs {
		currentActiveReviewers := make([]string, 0, len(pr.AssignedReviewers))
		reviewersToReplaceCount := 0

		for _, reviewerID := range pr.AssignedReviewers {
			isDeactivated := false
			for _, deactivatedID := range deactivatedUserIDs {
				if reviewerID == deactivatedID {
					isDeactivated = true
					reviewersToReplaceCount++
					break
				}
			}
			if !isDeactivated {
				currentActiveReviewers = append(currentActiveReviewers, reviewerID)
			}
		}

		if reviewersToReplaceCount == 0 {
			continue
		}

		author, err := s.user.GetUserById(ctx, pr.AuthorId)
		if err != nil {
			s.log.Error("service.TeamDeactivateUsers: failed to get author for reassignment logic (skipping PR)", slog.String("pr_id", pr.PullRequestId), slog.String("author_id", pr.AuthorId), slog.Any("error", err))
			failedCount++
			continue
		}

		activeCandidates, err := s.user.ListActiveMembersByTeam(ctx, author.TeamName, pr.AuthorId)
		if err != nil {
			s.log.Error("service.TeamDeactivateUsers: failed to list active members for replacement (skipping PR)", slog.String("team_name", author.TeamName), slog.Any("error", err))
			failedCount++
			continue
		}

		candidates := make([]domain.User, 0)
		assignedMap := make(map[string]bool)
		for _, id := range pr.AssignedReviewers {
			assignedMap[id] = true
		}

		for _, m := range activeCandidates {
			if !assignedMap[m.ID] {
				candidates = append(candidates, m)
			}
		}

		newReviewers := pickRandomUsers(candidates, reviewersToReplaceCount)

		if len(newReviewers) < reviewersToReplaceCount {
			s.log.Warn("service.TeamDeactivateUsers: not enough replacement candidates found", slog.String("pr_id", pr.PullRequestId), slog.Int("needed", reviewersToReplaceCount), slog.Int("available", len(candidates)))
			failedCount++
			continue
		}

		finalReviewers := currentActiveReviewers
		for _, newUser := range newReviewers {
			finalReviewers = append(finalReviewers, newUser.ID)
		}

		pr.AssignedReviewers = finalReviewers
		updatedPR, err := s.pr.UpdatePR(ctx, pr)
		if err != nil {
			s.log.Error("service.TeamDeactivateUsers: failed to update PR after reassignment", slog.String("pr_id", pr.PullRequestId), slog.Any("error", err))
			failedCount++
		} else {
			updatedPRs = append(updatedPRs, updatedPR)
			reassignedCount++
		}
	}

	return deactivatedUserIDs, updatedPRs, reassignedCount, failedCount, nil
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
