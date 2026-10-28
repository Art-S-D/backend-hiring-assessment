package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
)

type (
	// CreateAdFunc creates an ad owned by the calling user. It returns dao.ErrQuotaExceeded when the user has already used up their PublicationQuota.
	CreateAdFunc func(ctx context.Context, req CreateAdRequest) (domain.AdID, error)

	// PublicationQuota limits how many ads a user can publish through the single ad creation. The bulk import is not subject to it.
	PublicationQuota struct {
		// Max is how many ads a user can publish within Window.
		Max int
		// Window is the rolling window the quota applies to.
		Window time.Duration
	}

	// CreateAdRequest describes the ad to create. OwnerID is the authenticated user: a caller can never create an ad on behalf of someone else.
	CreateAdRequest struct {
		Title      string
		PriceCents int64
		PhotoURL   string
		OwnerID    domain.UserID
	}

	createAd struct {
		adDAO adsdao.DAO
		quota PublicationQuota
	}
)

// NewCreateAdFunc returns a CreateAdFunc backed by the given DAO and enforcing the given quota.
func NewCreateAdFunc(adDAO adsdao.DAO, quota PublicationQuota) CreateAdFunc {
	return createAd{adDAO: adDAO, quota: quota}.createAd
}

// createAd checks the quota, then creates the ad and records its publication in a single transaction, so an ad never exists without counting towards the quota.
// Concurrent requests from the same user can both pass the check and exceed the quota by a few ads: this is accepted for now.
func (uc createAd) createAd(ctx context.Context, req CreateAdRequest) (domain.AdID, error) {
	tx, err := uc.adDAO.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("unable to start transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			slog.ErrorContext(ctx, "unable to roll back ad creation transaction", "user_id", req.OwnerID, "error", err)
		}
	}()

	published, err := tx.CountPublicationsSince(ctx, req.OwnerID, uc.quota.Window)
	if err != nil {
		return 0, fmt.Errorf("unable to check publication quota: %w", err)
	}

	if published >= uc.quota.Max {
		return 0, fmt.Errorf("user already published %d ads in the last %s: %w", published, uc.quota.Window, adsdao.ErrQuotaExceeded)
	}

	adID, err := tx.CreateAd(ctx, adsdao.CreateAdRequest{
		Title:      req.Title,
		PriceCents: req.PriceCents,
		PhotoURL:   req.PhotoURL,
		OwnerID:    req.OwnerID,
	})
	if err != nil {
		return 0, fmt.Errorf("unable to create ad: %w", err)
	}

	if err := tx.RecordPublication(ctx, req.OwnerID, adID); err != nil {
		return 0, fmt.Errorf("unable to record publication of ad %s: %w", adID, err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("unable to commit creation of ad %s: %w", adID, err)
	}

	return adID, nil
}
