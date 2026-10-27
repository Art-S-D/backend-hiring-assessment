package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	mockdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao/mock"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
)

func TestDeleteAdByID(t *testing.T) {
	ctx := context.Background()
	owned := domain.Ad{ID: 42, Title: "bike", Price: 5000, UserID: "user-1"}

	t.Run("nominal", func(t *testing.T) {
		m := mockdao.New(t)
		m.On("Begin", ctx).Return(nil)
		m.On("GetAdByID", ctx, domain.AdID(42)).Return(owned, nil)
		m.On("DeleteAdByID", ctx, domain.AdID(42)).Return(nil)
		m.On("Commit").Return(nil)
		m.On("Rollback").Return(nil)

		err := usecase.NewDeleteAdByIDFunc(m)(ctx, 42, "user-1")

		require.NoError(t, err)
	})

	t.Run("error when the transaction cannot be opened", func(t *testing.T) {
		m := mockdao.New(t)
		dbErr := errors.New("db down")
		m.On("Begin", ctx).Return(dbErr)

		err := usecase.NewDeleteAdByIDFunc(m)(ctx, 42, "user-1")

		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("error no ad found", func(t *testing.T) {
		m := mockdao.New(t)
		m.On("Begin", ctx).Return(nil)
		m.On("GetAdByID", ctx, domain.AdID(99)).Return(domain.Ad{}, adsdao.ErrNotFound)
		m.On("Rollback").Return(nil)

		err := usecase.NewDeleteAdByIDFunc(m)(ctx, 99, "user-1")

		assert.ErrorIs(t, err, adsdao.ErrNotFound)
	})

	t.Run("error forbidden when the caller does not own the ad", func(t *testing.T) {
		m := mockdao.New(t)
		m.On("Begin", ctx).Return(nil)
		m.On("GetAdByID", ctx, domain.AdID(42)).Return(owned, nil)
		m.On("Rollback").Return(nil)

		err := usecase.NewDeleteAdByIDFunc(m)(ctx, 42, "user-2")

		assert.ErrorIs(t, err, adsdao.ErrForbidden)
		m.AssertNotCalled(t, "DeleteAdByID", ctx, domain.AdID(42))
		m.AssertNotCalled(t, "Commit")
	})

	t.Run("error from dao on delete", func(t *testing.T) {
		m := mockdao.New(t)
		dbErr := errors.New("connection reset")
		m.On("Begin", ctx).Return(nil)
		m.On("GetAdByID", ctx, domain.AdID(42)).Return(owned, nil)
		m.On("DeleteAdByID", ctx, domain.AdID(42)).Return(dbErr)
		m.On("Rollback").Return(nil)

		err := usecase.NewDeleteAdByIDFunc(m)(ctx, 42, "user-1")

		assert.ErrorIs(t, err, dbErr)
		m.AssertNotCalled(t, "Commit")
	})

	t.Run("error on commit", func(t *testing.T) {
		m := mockdao.New(t)
		commitErr := errors.New("commit failed")
		m.On("Begin", ctx).Return(nil)
		m.On("GetAdByID", ctx, domain.AdID(42)).Return(owned, nil)
		m.On("DeleteAdByID", ctx, domain.AdID(42)).Return(nil)
		m.On("Commit").Return(commitErr)
		m.On("Rollback").Return(nil)

		err := usecase.NewDeleteAdByIDFunc(m)(ctx, 42, "user-1")

		assert.ErrorIs(t, err, commitErr)
	})
}
