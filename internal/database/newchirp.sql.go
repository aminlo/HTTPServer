// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: newchirp.sql

package database

import (
	"context"

	"github.com/google/uuid"
)

const createChirp = `-- name: CreateChirp :one
INSERT INTO posts (id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(), NOW(), NOW(), $1, $2
)
RETURNING id, created_at, updated_at, body, user_id
`

type CreateChirpParams struct {
	Body   string
	UserID uuid.UUID
}

func (q *Queries) CreateChirp(ctx context.Context, arg CreateChirpParams) (Post, error) {
	row := q.db.QueryRowContext(ctx, createChirp, arg.Body, arg.UserID)
	var i Post
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Body,
		&i.UserID,
	)
	return i, err
}
