package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao/mock"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
)

func TestCreateAd(t *testing.T) {
	ctx := context.Background()

	ucReq := usecase.CreateAdRequest{
		Title:      "mountain bike",
		PriceCents: 15000,
		PhotoURL:   "https://example.com/bike.jpg",
		OwnerID:    "user-1",
	}
	daoReq := adsdao.CreateAdRequest{
		Title:      "mountain bike",
		PriceCents: 15000,
		PhotoURL:   "https://example.com/bike.jpg",
		OwnerID:    "user-1",
	}
	var owner domain.UserID = "user-1"
	quota := usecase.PublicationQuota{Max: 5, Window: 24 * time.Hour}

	t.Run("nominal", func(t *testing.T) {
		m := mock.New(t)
		m.On("Begin", ctx).Return(nil)
		m.On("CountPublicationsSince", ctx, owner, quota.Window).Return(quota.Max-1, nil)
		m.On("CreateAd", ctx, daoReq).Return(domain.AdID(7), nil)
		m.On("RecordPublication", ctx, owner, domain.AdID(7)).Return(nil)
		m.On("Commit").Return(nil)
		m.On("Rollback").Return(nil)

		got, err := usecase.NewCreateAdFunc(m, quota)(ctx, ucReq)

		require.NoError(t, err)
		assert.Equal(t, domain.AdID(7), got)
	})

	t.Run("quota exceeded", func(t *testing.T) {
		m := mock.New(t)
		m.On("Begin", ctx).Return(nil)
		m.On("CountPublicationsSince", ctx, owner, quota.Window).Return(quota.Max, nil)
		m.On("Rollback").Return(nil)

		_, err := usecase.NewCreateAdFunc(m, quota)(ctx, ucReq)

		assert.ErrorIs(t, err, adsdao.ErrQuotaExceeded)
		m.AssertNotCalled(t, "CreateAd")
		m.AssertNotCalled(t, "Commit")
	})

	t.Run("error when the transaction cannot be opened", func(t *testing.T) {
		m := mock.New(t)
		dbErr := errors.New("db down")
		m.On("Begin", ctx).Return(dbErr)

		_, err := usecase.NewCreateAdFunc(m, quota)(ctx, ucReq)

		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("error when the quota cannot be checked", func(t *testing.T) {
		m := mock.New(t)
		dbErr := errors.New("connection reset")
		m.On("Begin", ctx).Return(nil)
		m.On("CountPublicationsSince", ctx, owner, quota.Window).Return(0, dbErr)
		m.On("Rollback").Return(nil)

		_, err := usecase.NewCreateAdFunc(m, quota)(ctx, ucReq)

		assert.ErrorIs(t, err, dbErr)
		m.AssertNotCalled(t, "CreateAd")
	})

	t.Run("error from dao", func(t *testing.T) {
		m := mock.New(t)
		dbErr := errors.New("connection reset")
		m.On("Begin", ctx).Return(nil)
		m.On("CountPublicationsSince", ctx, owner, quota.Window).Return(0, nil)
		m.On("CreateAd", ctx, daoReq).Return(domain.AdID(0), dbErr)
		m.On("Rollback").Return(nil)

		_, err := usecase.NewCreateAdFunc(m, quota)(ctx, ucReq)

		require.Error(t, err)
		assert.ErrorIs(t, err, dbErr)
		m.AssertNotCalled(t, "Commit")
	})

	t.Run("rolls back the ad when its publication cannot be recorded", func(t *testing.T) {
		m := mock.New(t)
		dbErr := errors.New("connection reset")
		m.On("Begin", ctx).Return(nil)
		m.On("CountPublicationsSince", ctx, owner, quota.Window).Return(0, nil)
		m.On("CreateAd", ctx, daoReq).Return(domain.AdID(7), nil)
		m.On("RecordPublication", ctx, owner, domain.AdID(7)).Return(dbErr)
		m.On("Rollback").Return(nil)

		_, err := usecase.NewCreateAdFunc(m, quota)(ctx, ucReq)

		assert.ErrorIs(t, err, dbErr)
		m.AssertNotCalled(t, "Commit")
	})

	t.Run("error on commit", func(t *testing.T) {
		m := mock.New(t)
		dbErr := errors.New("serialization failure")
		m.On("Begin", ctx).Return(nil)
		m.On("CountPublicationsSince", ctx, owner, quota.Window).Return(0, nil)
		m.On("CreateAd", ctx, daoReq).Return(domain.AdID(7), nil)
		m.On("RecordPublication", ctx, owner, domain.AdID(7)).Return(nil)
		m.On("Commit").Return(dbErr)
		m.On("Rollback").Return(nil)

		_, err := usecase.NewCreateAdFunc(m, quota)(ctx, ucReq)

		assert.ErrorIs(t, err, dbErr)
	})
}
