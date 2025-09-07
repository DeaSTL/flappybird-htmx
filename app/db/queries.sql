-- name: CreateUser :exec
INSERT INTO users (id,name, created_at, updated_at, last_score, top_score) VALUES ( ?,?,?,?,?,?);

-- name: GetUserByName :one
SELECT * FROM users WHERE name = ?;

-- name: UpdateUserScore :exec
UPDATE users SET last_score = ?, top_score = ?, updated_at = ? WHERE name = ?;

-- name: GetTopScores :many
SELECT name, top_score FROM users ORDER BY top_score DESC LIMIT ?;

-- name: GetRecentScores :many
SELECT name, last_score, updated_at FROM users ORDER BY updated_at DESC LIMIT ?;
