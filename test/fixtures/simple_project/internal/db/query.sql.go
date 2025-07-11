package db

import (
	"context"
	"database/sql"
	"time"
)

type Queries struct {
	db DBTX
}

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

const createComment = `-- name: CreateComment :one
INSERT INTO comments (post_id, author_id, content) VALUES ($1, $2, $3) RETURNING id, post_id, author_id, content, created_at
`

type CreateCommentParams struct {
	PostID   int32  `json:"post_id"`
	AuthorID int32  `json:"author_id"`
	Content  string `json:"content"`
}

func (q *Queries) CreateComment(ctx context.Context, arg CreateCommentParams) (Comment, error) {
	row := q.db.QueryRowContext(ctx, createComment, arg.PostID, arg.AuthorID, arg.Content)
	var i Comment
	err := row.Scan(
		&i.ID,
		&i.PostID,
		&i.AuthorID,
		&i.Content,
		&i.CreatedAt,
	)
	return i, err
}

const createPost = `-- name: CreatePost :one
INSERT INTO posts (title, content, author_id) VALUES ($1, $2, $3) RETURNING id, title, content, author_id, created_at
`

type CreatePostParams struct {
	Title    string  `json:"title"`
	Content  *string `json:"content"`
	AuthorID int32   `json:"author_id"`
}

func (q *Queries) CreatePost(ctx context.Context, arg CreatePostParams) (Post, error) {
	row := q.db.QueryRowContext(ctx, createPost, arg.Title, arg.Content, arg.AuthorID)
	var i Post
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.Content,
		&i.AuthorID,
		&i.CreatedAt,
	)
	return i, err
}

const createUser = `-- name: CreateUser :one
INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at
`

type CreateUserParams struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRowContext(ctx, createUser, arg.Name, arg.Email)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Email,
		&i.CreatedAt,
	)
	return i, err
}

const getCommentsByPost = `-- name: GetCommentsByPost :many
SELECT c.id, c.content, c.author_id, c.created_at, u.name as author_name
FROM comments c JOIN users u ON c.author_id = u.id WHERE c.post_id = $1 ORDER BY c.created_at
`

type GetCommentsByPostRow struct {
	ID         int32     `json:"id"`
	Content    string    `json:"content"`
	AuthorID   int32     `json:"author_id"`
	CreatedAt  time.Time `json:"created_at"`
	AuthorName string    `json:"author_name"`
}

func (q *Queries) GetCommentsByPost(ctx context.Context, postID int32) ([]GetCommentsByPostRow, error) {
	rows, err := q.db.QueryContext(ctx, getCommentsByPost, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetCommentsByPostRow{}
	for rows.Next() {
		var i GetCommentsByPostRow
		if err := rows.Scan(
			&i.ID,
			&i.Content,
			&i.AuthorID,
			&i.CreatedAt,
			&i.AuthorName,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getPost = `-- name: GetPost :one
SELECT p.id, p.title, p.content, p.author_id, p.created_at, u.name as author_name 
FROM posts p JOIN users u ON p.author_id = u.id WHERE p.id = $1
`

type GetPostRow struct {
	ID         int32     `json:"id"`
	Title      string    `json:"title"`
	Content    *string   `json:"content"`
	AuthorID   int32     `json:"author_id"`
	CreatedAt  time.Time `json:"created_at"`
	AuthorName string    `json:"author_name"`
}

func (q *Queries) GetPost(ctx context.Context, id int32) (GetPostRow, error) {
	row := q.db.QueryRowContext(ctx, getPost, id)
	var i GetPostRow
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.Content,
		&i.AuthorID,
		&i.CreatedAt,
		&i.AuthorName,
	)
	return i, err
}

const getUser = `-- name: GetUser :one
SELECT id, name, email, created_at FROM users WHERE id = $1
`

func (q *Queries) GetUser(ctx context.Context, id int32) (User, error) {
	row := q.db.QueryRowContext(ctx, getUser, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Email,
		&i.CreatedAt,
	)
	return i, err
}

const listPostsByUser = `-- name: ListPostsByUser :many
SELECT id, title, content, author_id, created_at FROM posts WHERE author_id = $1 ORDER BY created_at DESC
`

func (q *Queries) ListPostsByUser(ctx context.Context, authorID int32) ([]Post, error) {
	rows, err := q.db.QueryContext(ctx, listPostsByUser, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Post{}
	for rows.Next() {
		var i Post
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Content,
			&i.AuthorID,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listUsers = `-- name: ListUsers :many
SELECT id, name, email, created_at FROM users ORDER BY created_at DESC
`

func (q *Queries) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := q.db.QueryContext(ctx, listUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []User{}
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Email,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}