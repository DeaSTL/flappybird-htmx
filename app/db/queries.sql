-- name: CreateUser :exec
INSERT INTO users (id,name, created_at, updated_at, last_score, top_score) VALUES ( ?,?,?,?,?,?);

-- name: GetUserByName :one
SELECT * FROM users WHERE name = ?;
