package service

import (
	"context"
	"database/sql"
	"github.com/naoyafurudono/sqlc-use-analysis/test/fixtures/simple_project/internal/db"
)

type PostService struct {
	queries *db.Queries
}

func NewPostService(database *sql.DB) *PostService {
	return &PostService{
		queries: db.New(database),
	}
}

func (s *PostService) CreatePost(ctx context.Context, title, content string, authorID int32) (*db.Post, error) {
	post, err := s.queries.CreatePost(ctx, db.CreatePostParams{
		Title:    title,
		Content:  &content,
		AuthorID: authorID,
	})
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (s *PostService) GetPost(ctx context.Context, id int32) (*db.GetPostRow, error) {
	post, err := s.queries.GetPost(ctx, id)
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (s *PostService) GetPostComments(ctx context.Context, postID int32) ([]db.GetCommentsByPostRow, error) {
	return s.queries.GetCommentsByPost(ctx, postID)
}

func (s *PostService) AddComment(ctx context.Context, postID, authorID int32, content string) (*db.Comment, error) {
	comment, err := s.queries.CreateComment(ctx, db.CreateCommentParams{
		PostID:   postID,
		AuthorID: authorID,
		Content:  content,
	})
	if err != nil {
		return nil, err
	}
	return &comment, nil
}