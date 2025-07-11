-- name: GetUser :one
SELECT id, name, email, created_at FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, name, email, created_at FROM users ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at;

-- name: GetPost :one
SELECT p.id, p.title, p.content, p.author_id, p.created_at, u.name as author_name 
FROM posts p JOIN users u ON p.author_id = u.id WHERE p.id = $1;

-- name: ListPostsByUser :many
SELECT id, title, content, author_id, created_at FROM posts WHERE author_id = $1 ORDER BY created_at DESC;

-- name: CreatePost :one
INSERT INTO posts (title, content, author_id) VALUES ($1, $2, $3) RETURNING id, title, content, author_id, created_at;

-- name: GetCommentsByPost :many
SELECT c.id, c.content, c.author_id, c.created_at, u.name as author_name
FROM comments c JOIN users u ON c.author_id = u.id WHERE c.post_id = $1 ORDER BY c.created_at;

-- name: CreateComment :one
INSERT INTO comments (post_id, author_id, content) VALUES ($1, $2, $3) RETURNING id, post_id, author_id, content, created_at;