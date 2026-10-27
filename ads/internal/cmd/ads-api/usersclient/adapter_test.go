package usersclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usersclient"
	usersPkg "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg/api"
)

func newAdapter(t *testing.T, mux *http.ServeMux) usersclient.Adapter {
	t.Helper()

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return usersclient.New(api.NewClient(srv.URL, 0))
}

func TestAdapterGetUserByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the user on success", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /v1/users/{userID}", func(rw http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "user-1", r.PathValue("userID"))
			_ = json.NewEncoder(rw).Encode(api.User{UserID: "user-1", Nickname: "alice"})
		})

		got, err := newAdapter(t, mux).GetUserByID(ctx, "user-1")

		require.NoError(t, err)
		assert.Equal(t, usersPkg.User{UserID: "user-1", Nickname: "alice"}, got)
	})

	t.Run("keeps ErrNotFound when the user does not exist", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /v1/users/{userID}", func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNotFound)
		})

		_, err := newAdapter(t, mux).GetUserByID(ctx, "unknown")

		require.ErrorIs(t, err, api.ErrNotFound)
	})
}

func TestAdapterListUsers(t *testing.T) {
	ctx := context.Background()

	t.Run("returns all users on success", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /v1/users", func(rw http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(rw).Encode([]api.User{
				{UserID: "user-1", Nickname: "alice"},
				{UserID: "user-2", Nickname: "bob"},
			})
		})

		got, err := newAdapter(t, mux).ListUsers(ctx)

		require.NoError(t, err)
		assert.Equal(t, []usersPkg.User{
			{UserID: "user-1", Nickname: "alice"},
			{UserID: "user-2", Nickname: "bob"},
		}, got)
	})

	t.Run("fails when the identity service errors", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /v1/users", func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusInternalServerError)
		})

		_, err := newAdapter(t, mux).ListUsers(ctx)

		require.Error(t, err)
	})
}
