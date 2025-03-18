--name RevokeRToken :none
UPDATE refresh_tokens
SET revoked_at = $2, updated_at = $3
WHERE token = $1;
