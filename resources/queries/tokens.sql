-- name: RevokeToken :exec
insert into revoked_tokens (token_id, expiry) values (?, ?);

-- name: TokenIsRevoked :one
select count(*) > 0 as is_revoked
from revoked_tokens
where token_id = ?
;

-- name: CleanRevokedToken :exec
delete from revoked_tokens
where token_id = ?
;

-- name: GetAllRevokedTokens :many
select *
from revoked_tokens
;

-- name: GetRevokedToken :one
select *
from revoked_tokens
where token_id = ?
;

