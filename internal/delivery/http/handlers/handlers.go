package handlers

import (
	"context"
	"errors"
	"fmt"

	api "github.com/3eLLenKa/test-avito/internal/delivery/http/gen"
	"github.com/3eLLenKa/test-avito/internal/domain"
)

type Service interface {
	PullRequestCreate(ctx context.Context, prId, prName, authorId string) (*domain.PullRequest, error)
	PullRequestMerge(ctx context.Context, prId string) (*domain.PullRequest, error)
	PullRequestReassign(ctx context.Context, prId, oldUserId string) (*domain.PullRequest, string, error)
	TeamAdd(ctx context.Context, teamName string, members []domain.User) (*domain.Team, error)
	TeamGet(ctx context.Context, teamName string) (*domain.Team, error)
	UsersGetReview(ctx context.Context, userId string) ([]*domain.PullRequest, error)
	SetUserActive(ctx context.Context, userId string, isActive bool) (*domain.User, error)
}

type Handlers struct {
	svc Service
}

func NewHandlers(svc Service) api.StrictServerInterface {
	return &Handlers{svc: svc}
}

func (h *Handlers) PostPullRequestCreate(ctx context.Context, request api.PostPullRequestCreateRequestObject) (api.PostPullRequestCreateResponseObject, error) {
	pr, err := h.svc.PullRequestCreate(ctx,
		request.Body.PullRequestId,
		request.Body.PullRequestName,
		request.Body.AuthorId,
	)

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPRExists):
			return api.PostPullRequestCreate409JSONResponse(
				errorResponse(api.ErrorResponseErrorCodePREXISTS, "PR id already exists"),
			), nil

		case errors.Is(err, domain.ErrUserNotFound):
			return api.PostPullRequestCreate404JSONResponse(
				errorResponse(api.ErrorResponseErrorCodeNOTFOUND, "author not found"),
			), nil
		case errors.Is(err, domain.ErrTeamNotFound):
			return api.PostPullRequestCreate404JSONResponse(
				errorResponse(api.ErrorResponseErrorCodeNOTFOUND, "author's team not found"),
			), nil
		default:
			return nil, err
		}
	}

	return api.PostPullRequestCreate201JSONResponse{
		Pr: &api.PullRequest{
			PullRequestId:     pr.PullRequestId,
			PullRequestName:   pr.PullRequestName,
			AuthorId:          pr.AuthorId,
			AssignedReviewers: pr.AssignedReviewers,
			MergedAt:          pr.MergedAt,
			CreatedAt:         pr.CreatedAt,
			Status:            api.PullRequestStatus(pr.Status),
		},
	}, nil
}

func (h *Handlers) PostPullRequestMerge(ctx context.Context, request api.PostPullRequestMergeRequestObject) (api.PostPullRequestMergeResponseObject, error) {
	pr, err := h.svc.PullRequestMerge(ctx, request.Body.PullRequestId)
	if err != nil {
		if errors.Is(err, domain.ErrPRNotFound) {
			return api.PostPullRequestMerge404JSONResponse(
				errorResponse(api.ErrorResponseErrorCodeNOTFOUND, "pull request not found"),
			), nil
		}
		return nil, err
	}

	return api.PostPullRequestMerge200JSONResponse{
		Pr: &api.PullRequest{
			PullRequestId:     pr.PullRequestId,
			PullRequestName:   pr.PullRequestName,
			AuthorId:          pr.AuthorId,
			AssignedReviewers: pr.AssignedReviewers,
			MergedAt:          pr.MergedAt,
			CreatedAt:         pr.CreatedAt,
			Status:            api.PullRequestStatus(pr.Status),
		},
	}, nil
}

func (h *Handlers) PostPullRequestReassign(ctx context.Context, request api.PostPullRequestReassignRequestObject) (api.PostPullRequestReassignResponseObject, error) {
	pr, replacedBy, err := h.svc.PullRequestReassign(ctx, request.Body.PullRequestId, request.Body.OldUserId)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPRNotFound), errors.Is(err, domain.ErrUserNotFound):
			return api.PostPullRequestReassign404JSONResponse(
				errorResponse(api.ErrorResponseErrorCodeNOTFOUND, "PR or user not found"),
			), nil
		case errors.Is(err, domain.ErrPRMerged):
			return api.PostPullRequestReassign409JSONResponse(
				errorResponse(api.ErrorResponseErrorCodePRMERGED, "cannot reassign merged PR"),
			), nil
		case errors.Is(err, domain.ErrNotAssigned):
			return api.PostPullRequestReassign409JSONResponse(
				errorResponse(api.ErrorResponseErrorCodeNOTASSIGNED, "reviewer is not assigned to PR"),
			), nil
		case errors.Is(err, domain.ErrNoCandidate):
			return api.PostPullRequestReassign409JSONResponse(
				errorResponse(api.ErrorResponseErrorCodeNOCANDIDATE, "no active replacement candidate in team"),
			), nil
		}
		return nil, err
	}

	return api.PostPullRequestReassign200JSONResponse{
		Pr: api.PullRequest{
			PullRequestId:     pr.PullRequestId,
			PullRequestName:   pr.PullRequestName,
			AuthorId:          pr.AuthorId,
			AssignedReviewers: pr.AssignedReviewers,
			MergedAt:          pr.MergedAt,
			CreatedAt:         pr.CreatedAt,
			Status:            api.PullRequestStatus(pr.Status),
		},
		ReplacedBy: replacedBy,
	}, nil
}

func (h *Handlers) PostUsersSetIsActive(ctx context.Context, request api.PostUsersSetIsActiveRequestObject) (api.PostUsersSetIsActiveResponseObject, error) {
	user, err := h.svc.SetUserActive(ctx, request.Body.UserId, request.Body.IsActive)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return api.PostUsersSetIsActive404JSONResponse(
				errorResponse(api.ErrorResponseErrorCodeNOTFOUND, "user not found"),
			), nil
		}
		return nil, fmt.Errorf("cannot update status: %w", err)
	}

	return api.PostUsersSetIsActive200JSONResponse{
		User: &api.User{
			UserId:   user.ID,
			Username: user.Name,
			TeamName: user.TeamName,
			IsActive: user.IsActive,
		},
	}, nil
}

func (h *Handlers) PostTeamAdd(ctx context.Context, request api.PostTeamAddRequestObject) (api.PostTeamAddResponseObject, error) {
	dMembers := []domain.User{}
	for _, m := range request.Body.Members {
		dMembers = append(dMembers, domain.User{
			ID:       m.UserId,
			Name:     m.Username,
			IsActive: m.IsActive,
			TeamName: request.Body.TeamName,
		})
	}

	team, err := h.svc.TeamAdd(ctx, request.Body.TeamName, dMembers)
	if err != nil {
		if errors.Is(err, domain.ErrTeamExists) {
			return api.PostTeamAdd400JSONResponse(
				errorResponse(api.ErrorResponseErrorCodeTEAMEXISTS, "team already exists"),
			), nil
		}
		return nil, err
	}

	members := []api.TeamMember{}
	for _, m := range team.Members {
		members = append(members, api.TeamMember{
			IsActive: m.IsActive,
			UserId:   m.ID,
			Username: m.Name,
		})
	}

	return api.PostTeamAdd201JSONResponse{
		Team: &api.Team{
			TeamName: team.Name,
			Members:  members,
		},
	}, nil
}

func (h *Handlers) GetTeamGet(ctx context.Context, request api.GetTeamGetRequestObject) (api.GetTeamGetResponseObject, error) {
	team, err := h.svc.TeamGet(ctx, request.Params.TeamName)
	if err != nil {
		if errors.Is(err, domain.ErrTeamNotFound) {
			return api.GetTeamGet404JSONResponse(
				errorResponse(api.ErrorResponseErrorCodeNOTFOUND, "team not found"),
			), nil
		}
		return nil, err
	}

	members := []api.TeamMember{}
	for _, m := range team.Members {
		members = append(members, api.TeamMember{
			IsActive: m.IsActive,
			UserId:   m.ID,
			Username: m.Name,
		})
	}

	return api.GetTeamGet200JSONResponse{
		TeamName: team.Name,
		Members:  members,
	}, nil
}

func (h *Handlers) GetUsersGetReview(
	ctx context.Context,
	request api.GetUsersGetReviewRequestObject,
) (api.GetUsersGetReviewResponseObject, error) {

	prs, err := h.svc.UsersGetReview(ctx, request.Params.UserId)
	if err != nil {
		return nil, fmt.Errorf("cannot get reviews: %w", err)
	}

	resp := []api.PullRequestShort{}
	for _, pr := range prs {
		resp = append(resp, api.PullRequestShort{
			AuthorId:        pr.AuthorId,
			PullRequestId:   pr.PullRequestId,
			PullRequestName: pr.PullRequestName,
			Status:          api.PullRequestShortStatus(pr.Status),
		})
	}

	return api.GetUsersGetReview200JSONResponse{
		UserId:       request.Params.UserId,
		PullRequests: resp,
	}, nil
}

// вспомогательные функции:

func errorResponse(code api.ErrorResponseErrorCode, message string) api.ErrorResponse {
	return api.ErrorResponse{
		Error: struct {
			Code    api.ErrorResponseErrorCode "json:\"code\""
			Message string                     "json:\"message\""
		}{
			Code:    code,
			Message: message,
		},
	}
}
