package usersclient

import (
	"context"
	"fmt"

	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/model"
	usersPkg "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg/api"
)

// Adapter makes api.Client satisfy usersPkg.UserClient, which it does not implement.
// It takes a plain string identifier and returns its own User type.
// The proper fix belongs in /users, but I was explicitly told not to touch it.
type Adapter struct {
	client *api.Client
}

var _ usersPkg.UserClient = Adapter{}

// New returns an Adapter wrapping the given client.
func New(client *api.Client) Adapter {
	return Adapter{client: client}
}

// ListUsers returns all users registered in the identity service.
func (a Adapter) ListUsers(ctx context.Context) ([]usersPkg.User, error) {
	users, err := a.client.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to list users: %w", err)
	}

	converted := make([]usersPkg.User, 0, len(users))
	for _, user := range users {
		converted = append(converted, toUser(user))
	}

	return converted, nil
}

// GetUserByID returns the user with the given ID from the identity service.
func (a Adapter) GetUserByID(ctx context.Context, userID model.UserID) (usersPkg.User, error) {
	user, err := a.client.GetUserByID(ctx, userID.String())
	if err != nil {
		return usersPkg.User{}, fmt.Errorf("unable to get user %s: %w", userID, err)
	}

	return toUser(user), nil
}

func toUser(user api.User) usersPkg.User {
	return usersPkg.User{UserID: user.UserID, Nickname: user.Nickname}
}
