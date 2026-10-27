package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	mockdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao/mock"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	usersPkg "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
	usersAPI "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg/api"
	mockusers "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg/mock"
)

func TestGetAdByID(t *testing.T) {
	ctx := context.Background()

	t.Run("nominal", func(t *testing.T) {
		m := mockdao.New(t)
		users := mockusers.New(t)
		ad := domain.Ad{ID: 42, Title: "bike", Price: 5000, UserID: "user-1"}
		m.On("GetAdByID", ctx, domain.AdID(42)).Return(ad, nil)
		users.On("GetUserByID", ctx, ad.UserID).Return(usersPkg.User{UserID: ad.UserID, Nickname: "alice"}, nil)

		got, err := usecase.NewGetAdByIDFunc(m, users)(ctx, 42)

		require.NoError(t, err)
		assert.Equal(t, usecase.GetAdByIDResponse{Ad: ad, UserName: "alice"}, got)
	})

	t.Run("error no ad found", func(t *testing.T) {
		m := mockdao.New(t)
		m.On("GetAdByID", ctx, mock.Anything).Return(domain.Ad{}, adsdao.ErrNotFound)

		_, err := usecase.NewGetAdByIDFunc(m, mockusers.New(t))(ctx, 99)

		require.Error(t, err)
		assert.ErrorIs(t, err, adsdao.ErrNotFound)
	})

	t.Run("error from dao", func(t *testing.T) {
		m := mockdao.New(t)
		dbErr := errors.New("connection reset")
		m.On("GetAdByID", ctx, mock.Anything).Return(domain.Ad{}, dbErr)

		_, err := usecase.NewGetAdByIDFunc(m, mockusers.New(t))(ctx, 1)

		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("empty username when the owner does not exist", func(t *testing.T) {
		m := mockdao.New(t)
		users := mockusers.New(t)
		ad := domain.Ad{ID: 1, UserID: "deleted-user"}
		m.On("GetAdByID", ctx, mock.Anything).Return(ad, nil)
		users.On("GetUserByID", ctx, ad.UserID).Return(usersPkg.User{}, fmt.Errorf("wrapped: %w", usersAPI.ErrNotFound))

		got, err := usecase.NewGetAdByIDFunc(m, users)(ctx, 1)

		require.NoError(t, err)
		assert.Equal(t, usecase.GetAdByIDResponse{Ad: ad, UserName: ""}, got)
	})

	t.Run("error from users service", func(t *testing.T) {
		m := mockdao.New(t)
		users := mockusers.New(t)
		usersErr := errors.New("identity service answered 503")
		m.On("GetAdByID", ctx, mock.Anything).Return(domain.Ad{ID: 1, UserID: "user-1"}, nil)
		users.On("GetUserByID", ctx, mock.Anything).Return(usersPkg.User{}, usersErr)

		_, err := usecase.NewGetAdByIDFunc(m, users)(ctx, 1)

		assert.ErrorIs(t, err, usersErr)
	})
}
