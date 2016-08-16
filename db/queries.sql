-- name: load-user-by-token
SELECT * from users where token = $1
