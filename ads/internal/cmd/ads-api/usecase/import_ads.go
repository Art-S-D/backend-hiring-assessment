package usecase

import (
	"context"
	"fmt"
	"log/slog"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
)

type (
	// ImportAdsFunc creates a batch of ads owned by the calling user. The import is all-or-nothing: either every ad is created, or none is, so a failed import can be retried without creating duplicates.
	ImportAdsFunc func(ctx context.Context, req ImportAdsRequest) ([]domain.AdID, error)

	// ImportAdsRequest describes the ads to import. OwnerID is the authenticated user: a caller can never import ads on behalf of someone else.
	ImportAdsRequest struct {
		Ads     []ImportedAd
		OwnerID domain.UserID
	}

	// ImportedAd describes a single ad of an import.
	ImportedAd struct {
		Title      string
		PriceCents int64
		PhotoURL   string
	}

	importAds struct {
		adDAO adsdao.DAO
	}
)

// NewImportAdsFunc returns an ImportAdsFunc backed by the given DAO.
func NewImportAdsFunc(adDAO adsdao.DAO) ImportAdsFunc {
	return importAds{adDAO: adDAO}.importAds
}

func (uc importAds) importAds(ctx context.Context, req ImportAdsRequest) ([]domain.AdID, error) {
	ids := make([]domain.AdID, 0, len(req.Ads))
	if len(req.Ads) == 0 {
		return ids, nil
	}

	tx, err := uc.adDAO.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to start transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			slog.ErrorContext(ctx, "unable to roll back import transaction", "user_id", req.OwnerID, "error", err)
		}
	}()

	for i, ad := range req.Ads {
		id, err := tx.CreateAd(ctx, adsdao.CreateAdRequest{
			Title:      ad.Title,
			PriceCents: ad.PriceCents,
			PhotoURL:   ad.PhotoURL,
			OwnerID:    req.OwnerID,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to import ad #%d: %w", i, err)
		}

		ids = append(ids, id)
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("unable to commit import of %d ads: %w", len(ids), err)
	}

	return ids, nil
}
