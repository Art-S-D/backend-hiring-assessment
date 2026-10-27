package usecase

import (
	"context"
	"fmt"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/model"
)

type (
	// DeleteAdByIDFunc deletes an ad owned by the caller. It fails with ErrNotFound when no ad matches the identifier, and with ErrForbidden when the caller does not own the ad.
	DeleteAdByIDFunc func(ctx context.Context, id domain.AdID, callerID model.UserID) error

	deleteAdByID struct {
		adDAO adsdao.DAO
	}
)

// NewDeleteAdByIDFunc returns a DeleteAdByIDFunc backed by the given DAO.
func NewDeleteAdByIDFunc(adDAO adsdao.DAO) DeleteAdByIDFunc {
	return deleteAdByID{adDAO: adDAO}.deleteAdByID
}

func (uc deleteAdByID) deleteAdByID(ctx context.Context, id domain.AdID, callerID model.UserID) error {
	tx, err := uc.adDAO.Begin(ctx)
	if err != nil {
		return fmt.Errorf("unable to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	ad, err := tx.GetAdByID(ctx, id)
	if err != nil {
		return fmt.Errorf("unable to get ad %s: %w", id, err)
	}
	if ad.UserID != callerID {
		return fmt.Errorf("user %s cannot delete an ad owned by %s: %w", callerID, ad.UserID, adsdao.ErrForbidden)
	}

	err = tx.DeleteAdByID(ctx, id)
	if err != nil {
		return fmt.Errorf("unable to delete ad %s: %w", id, err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("unable to commit deletion of ad %s: %w", id, err)
	}
	return nil
}
