-- name: GetRToken :one
SELECT * FROM refresh_tokens WHERE token = $1;