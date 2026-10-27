package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/middleware"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/model"
)

// newDeleteAdServer mounts the handler behind RequireAuth, as the router does.
func newDeleteAdServer(fn usecase.DeleteAdByIDFunc) *httptest.Server {
	mux := http.NewServeMux()
	mux.Handle("DELETE /ads/{adID}", middleware.RequireAuth(handlerDeleteAd(fn)))

	return httptest.NewServer(mux)
}

// helper to send DELETE http requests
// since there is no method for this in the standard net/http library
func doDelete(t *testing.T, url string, userID model.UserID) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)
	if userID != "" {
		req.Header.Set("Authorization", string(userID))
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

func TestHandlerDeleteAd(t *testing.T) {
	t.Run("returns 204 with empty body on success", func(t *testing.T) {
		srv := newDeleteAdServer(func(_ context.Context, _ domain.AdID, _ model.UserID) error {
			return nil
		})
		defer srv.Close()

		resp := doDelete(t, srv.URL+"/ads/42", "user-1")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Empty(t, body)
	})

	t.Run("returns 401 without Authorization header", func(t *testing.T) {
		srv := newDeleteAdServer(func(_ context.Context, _ domain.AdID, _ model.UserID) error {
			t.Error("usecase should not be called")
			return nil
		})
		defer srv.Close()

		resp := doDelete(t, srv.URL+"/ads/42", "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("returns 400 when adID is not a number", func(t *testing.T) {
		srv := newDeleteAdServer(func(_ context.Context, _ domain.AdID, _ model.UserID) error {
			t.Error("usecase should not be called")
			return nil
		})
		defer srv.Close()

		resp := doDelete(t, srv.URL+"/ads/not-a-number", "user-1")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 404 when ad does not exist", func(t *testing.T) {
		srv := newDeleteAdServer(func(_ context.Context, _ domain.AdID, _ model.UserID) error {
			return adsdao.ErrNotFound
		})
		defer srv.Close()

		resp := doDelete(t, srv.URL+"/ads/99", "user-1")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 403 when the caller does not own the ad", func(t *testing.T) {
		srv := newDeleteAdServer(func(_ context.Context, _ domain.AdID, _ model.UserID) error {
			return fmt.Errorf("unable to delete ad: %w", adsdao.ErrForbidden)
		})
		defer srv.Close()

		resp := doDelete(t, srv.URL+"/ads/42", "user-2")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("returns 500 on unexpected usecase error", func(t *testing.T) {
		srv := newDeleteAdServer(func(_ context.Context, _ domain.AdID, _ model.UserID) error {
			return errors.New("db down")
		})
		defer srv.Close()

		resp := doDelete(t, srv.URL+"/ads/1", "user-1")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("passes the ad ID and caller ID to the usecase", func(t *testing.T) {
		var (
			receivedID     domain.AdID
			receivedCaller model.UserID
		)
		srv := newDeleteAdServer(func(_ context.Context, id domain.AdID, callerID model.UserID) error {
			receivedID = id
			receivedCaller = callerID
			return nil
		})
		defer srv.Close()

		resp := doDelete(t, srv.URL+"/ads/7", "user-1")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Equal(t, domain.AdID(7), receivedID)
		assert.Equal(t, model.UserID("user-1"), receivedCaller)
	})
}
