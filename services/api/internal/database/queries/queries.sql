-- name: PingDatabase :one
SELECT 1 as alive;

-- name: GetUserByID :one
SELECT id, username, username_normalized, created_at, deleted_at, onboarding_completed_at
FROM users
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (
    username,
    username_normalized
) VALUES (
    $1,
    $2
) RETURNING id, username, username_normalized, created_at, deleted_at, onboarding_completed_at;
