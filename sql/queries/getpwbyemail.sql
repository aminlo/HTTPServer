-- name: GetPwByEmail :one
SELECT * FROM users WHERE email = $1;