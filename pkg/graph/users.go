package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/nais/console/pkg/dbmodels"
	"github.com/nais/console/pkg/graph/model"
)

func (r *mutationResolver) CreateUser(ctx context.Context, input model.CreateUserInput) (*dbmodels.User, error) {
	u := &dbmodels.User{
		Email: input.Email,
		Name:  &input.Name,
	}
	err := r.createDB(ctx, u)
	return u, err
}

func (r *mutationResolver) UpdateUser(ctx context.Context, input model.UpdateUserInput) (*dbmodels.User, error) {
	u := &dbmodels.User{
		Model: dbmodels.Model{
			ID: input.ID,
		},
		Email: input.Email,
		Name:  input.Name,
	}
	err := r.updateDB(ctx, u)
	return u, err
}

func (r *queryResolver) Users(ctx context.Context, input *model.QueryUserInput) (*model.Users, error) {
	users := make([]*dbmodels.User, 0)
	pagination, err := r.paginatedQuery(ctx, input, &dbmodels.User{}, &users)
	return &model.Users{
		Pagination: pagination,
		Nodes:      users,
	}, err
}
