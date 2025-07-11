package handler

import (
	"context"
	"strconv"
	"github.com/naoyafurudono/sqlc-use-analysis/test/fixtures/simple_project/internal/service"
)

type PostHandler struct {
	postService *service.PostService
	userService *service.UserService
}

func NewPostHandler(postService *service.PostService, userService *service.UserService) *PostHandler {
	return &PostHandler{
		postService: postService,
		userService: userService,
	}
}

func (h *PostHandler) CreatePost(ctx context.Context, title, content string, authorIDStr string) error {
	authorID, err := strconv.Atoi(authorIDStr)
	if err != nil {
		return err
	}
	
	// Verify author exists
	author, err := h.userService.GetUser(ctx, int32(authorID))
	if err != nil {
		return err
	}
	
	post, err := h.postService.CreatePost(ctx, title, content, author.ID)
	if err != nil {
		return err
	}
	
	_ = post // Use the post for something
	return nil
}

func (h *PostHandler) GetPostWithComments(ctx context.Context, postIDStr string) error {
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		return err
	}
	
	post, err := h.postService.GetPost(ctx, int32(postID))
	if err != nil {
		return err
	}
	
	comments, err := h.postService.GetPostComments(ctx, post.ID)
	if err != nil {
		return err
	}
	
	_ = comments // Use the comments for something
	return nil
}

func (h *PostHandler) AddComment(ctx context.Context, postIDStr, authorIDStr, content string) error {
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		return err
	}
	
	authorID, err := strconv.Atoi(authorIDStr)
	if err != nil {
		return err
	}
	
	// Verify post exists
	post, err := h.postService.GetPost(ctx, int32(postID))
	if err != nil {
		return err
	}
	
	// Verify author exists
	author, err := h.userService.GetUser(ctx, int32(authorID))
	if err != nil {
		return err
	}
	
	comment, err := h.postService.AddComment(ctx, post.ID, author.ID, content)
	if err != nil {
		return err
	}
	
	_ = comment // Use the comment for something
	return nil
}