package handler

import (
	"context"
	"strconv"
	"github.com/naoyafurudono/sqlc-use-analysis/test/fixtures/simple_project/internal/service"
)

type UserHandler struct {
	userService *service.UserService
	postService *service.PostService
}

func NewUserHandler(userService *service.UserService, postService *service.PostService) *UserHandler {
	return &UserHandler{
		userService: userService,
		postService: postService,
	}
}

func (h *UserHandler) CreateUser(ctx context.Context, name, email string) error {
	user, err := h.userService.CreateUser(ctx, name, email)
	if err != nil {
		return err
	}
	_ = user // Use the user for something
	return nil
}

func (h *UserHandler) GetUserProfile(ctx context.Context, userIDStr string) error {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return err
	}
	
	user, err := h.userService.GetUser(ctx, int32(userID))
	if err != nil {
		return err
	}
	
	posts, err := h.userService.GetUserPosts(ctx, user.ID)
	if err != nil {
		return err
	}
	
	_ = posts // Use the posts for something
	return nil
}

func (h *UserHandler) ListAllUsers(ctx context.Context) error {
	users, err := h.userService.ListUsers(ctx)
	if err != nil {
		return err
	}
	
	_ = users // Use the users for something
	return nil
}