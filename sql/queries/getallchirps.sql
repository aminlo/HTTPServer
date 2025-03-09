-- name: GetAllChirps :many
SELECT * 
FROM posts 
ORDER BY created_at ASC;