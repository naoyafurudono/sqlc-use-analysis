package service

import (
	"context"
	"database/sql"
	"github.com/naoyafurudono/sqlc-use-analysis/test/fixtures/simple_project/internal/db"
)

type UserService struct {
	queries *db.Queries
}

func NewUserService(database *sql.DB) *UserService {
	return &UserService{
		queries: db.New(database),
	}
}

func (s *UserService) CreateUser(ctx context.Context, name, email string) (*db.User, error) {
	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Name:  name,
		Email: email,
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) GetUser(ctx context.Context, id int32) (*db.User, error) {
	user, err := s.queries.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) ListUsers(ctx context.Context) ([]db.User, error) {
	return s.queries.ListUsers(ctx)
}

func (s *UserService) GetUserPosts(ctx context.Context, userID int32) ([]db.Post, error) {
	return s.queries.ListPostsByUser(ctx, userID)
}