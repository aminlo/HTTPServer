-- name: GetPost :one
SELECT * FROM posts WHERE id = $1;