package db

import (
	"time"
)

type User struct {
	ID        int32     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Post struct {
	ID        int32     `json:"id"`
	Title     string    `json:"title"`
	Content   *string   `json:"content"`
	AuthorID  int32     `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Comment struct {
	ID        int32     `json:"id"`
	PostID    int32     `json:"post_id"`
	AuthorID  int32     `json:"author_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}