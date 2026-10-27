package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	usersPkg "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
	usersAPI "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg/api"
)

type (
	// GetAdByIDFunc returns a single ad along with its owner's username, fetched from the users service.
	// It fails with adsdao.ErrNotFound when no ad matches the identifier. When the owner does not exist the username is left empty;
	// any other error from the users service makes it fail.
	GetAdByIDFunc func(ctx context.Context, id domain.AdID) (GetAdByIDResponse, error)

	// GetAdByIDResponse is the requested ad, enriched with the name of the user who owns it.
	GetAdByIDResponse struct {
		domain.Ad
		UserName string
	}

	getAdByID struct {
		adDAO adsdao.DAO
		users usersPkg.UserClient
	}
)

// NewGetAdByIDFunc returns a GetAdByIDFunc backed by the given DAO and users client.
func NewGetAdByIDFunc(adDAO adsdao.DAO, users usersPkg.UserClient) GetAdByIDFunc {
	return getAdByID{adDAO: adDAO, users: users}.getAdByID
}

func (uc getAdByID) getAdByID(ctx context.Context, id domain.AdID) (GetAdByIDResponse, error) {
	found, err := uc.adDAO.GetAdByID(ctx, id)
	if err != nil {
		return GetAdByIDResponse{}, fmt.Errorf("unable to get ad %s: %w", id, err)
	}

	user, err := uc.users.GetUserByID(ctx, found.UserID)
	switch {
	case errors.Is(err, usersAPI.ErrNotFound):
		// A missing owner (e.g. a deleted account) should not hide the ad: the username is left empty.
		slog.WarnContext(ctx, "ad owner not found in users service", "ad_id", id, "user_id", found.UserID)

		return GetAdByIDResponse{Ad: found}, nil
	case err != nil:
		return GetAdByIDResponse{}, fmt.Errorf("unable to get user %s for ad %s: %w", found.UserID, id, err)
	}

	return GetAdByIDResponse{Ad: found, UserName: user.Nickname}, nil
}
