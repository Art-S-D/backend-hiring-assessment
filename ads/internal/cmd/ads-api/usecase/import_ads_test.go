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

func TestImportAds(t *testing.T) {
	ctx := context.Background()

	ucReq := usecase.ImportAdsRequest{
		OwnerID: "user-1",
		Ads: []usecase.ImportedAd{
			{Title: "bike", PriceCents: 5000, PhotoURL: "https://example.com/bike.jpg"},
			{Title: "scooter", PriceCents: 8000, PhotoURL: "https://example.com/scooter.jpg"},
		},
	}
	bikeReq := adsdao.CreateAdRequest{Title: "bike", PriceCents: 5000, PhotoURL: "https://example.com/bike.jpg", OwnerID: "user-1"}
	scooterReq := adsdao.CreateAdRequest{Title: "scooter", PriceCents: 8000, PhotoURL: "https://example.com/scooter.jpg", OwnerID: "user-1"}

	t.Run("nominal", func(t *testing.T) {
		m := mockdao.New(t)
		m.On("Begin", ctx).Return(nil)
		m.On("CreateAd", ctx, bikeReq).Return(domain.AdID(10), nil)
		m.On("CreateAd", ctx, scooterReq).Return(domain.AdID(11), nil)
		m.On("Commit").Return(nil)
		m.On("Rollback").Return(nil)

		got, err := usecase.NewImportAdsFunc(m)(ctx, ucReq)

		require.NoError(t, err)
		assert.Equal(t, []domain.AdID{10, 11}, got)
	})

	t.Run("no ads does not open a transaction", func(t *testing.T) {
		m := mockdao.New(t)

		got, err := usecase.NewImportAdsFunc(m)(ctx, usecase.ImportAdsRequest{OwnerID: "user-1"})

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("error when the transaction cannot be opened", func(t *testing.T) {
		m := mockdao.New(t)
		dbErr := errors.New("db down")
		m.On("Begin", ctx).Return(dbErr)

		_, err := usecase.NewImportAdsFunc(m)(ctx, ucReq)

		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("rolls back every ad when one fails", func(t *testing.T) {
		m := mockdao.New(t)
		dbErr := errors.New("connection reset")
		m.On("Begin", ctx).Return(nil)
		m.On("CreateAd", ctx, bikeReq).Return(domain.AdID(10), nil)
		m.On("CreateAd", ctx, scooterReq).Return(domain.AdID(0), dbErr)
		m.On("Rollback").Return(nil)

		got, err := usecase.NewImportAdsFunc(m)(ctx, ucReq)

		assert.ErrorIs(t, err, dbErr)
		assert.Nil(t, got)
		m.AssertNotCalled(t, "Commit")
	})

	t.Run("error on commit", func(t *testing.T) {
		m := mockdao.New(t)
		dbErr := errors.New("serialization failure")
		m.On("Begin", ctx).Return(nil)
		m.On("CreateAd", ctx, bikeReq).Return(domain.AdID(10), nil)
		m.On("CreateAd", ctx, scooterReq).Return(domain.AdID(11), nil)
		m.On("Commit").Return(dbErr)
		m.On("Rollback").Return(nil)

		got, err := usecase.NewImportAdsFunc(m)(ctx, ucReq)

		assert.ErrorIs(t, err, dbErr)
		assert.Nil(t, got)
	})
}
