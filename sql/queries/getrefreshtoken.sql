-- name: GetRToken :one
SELECT * FROM refresh_tokens WHERE id = $1;