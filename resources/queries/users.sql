-- name: GetUserCount :one
select count(*)
from users
;

-- name: GetAllUsers :many
select *
from users
;

-- name: GetUser :one
select *
from users
where user_id = ?
limit 1
;

-- name: GetUserSecretsFromEmail :one
select user_secrets.*
from users
join user_secrets on users.user_id = user_secrets.user_id
where users.email = ?
;

-- name: GetUserSecretsFromID :one
select *
from user_secrets
where user_id = ?
limit 1
;


-- name: DeleteUser :exec
delete from users
where user_id = ?
;

-- name: GetUserRoles :many
select role
from user_roles
where user_id = ?
;

-- name: UserHasRole :one
select count(*) > 0 as is_allowed
from user_roles
where user_id = ? and role in (sqlc.slice('roles'))
;

-- name: AppendUserRole :exec
insert into user_roles (user_id, role)
values (?, ?)
on conflict do nothing;

-- name: AppendUserRoleFromEmail :exec
insert into user_roles (user_id, role)
select user_id, ?
from users
where email = ?
on conflict(user_id, role) do nothing;

-- name: RevokeUserRole :exec
delete from user_roles
where user_id = ? and role = ?
;

-- name: RevokeUserRoleFromEmail :exec
delete from user_roles
where user_id = (select user_id from users where email = ?) and role = ?
;

-- name: UserWithEmailExists :one
select count(*) > 0 as user_exists
from users
where email = ?
;

-- name: UserExists :one
select count(*) > 0 as user_exists
from users
where user_id = ?
;

-- name: UpdateUserSecret :exec
update user_secrets
set salt = ?, hash = ?
where user_id = ?
;

-- name: UpdateUserModified :one
update users
set updated_at = CURRENT_TIMESTAMP
returning updated_at;

-- name: UpdateUser :one
update users
set name = ?, email = ?, updated_at = CURRENT_TIMESTAMP
where user_id = ?
returning updated_at
;

-- name: NewUser :one
insert into users (name, email) values (?, ?) returning *;

-- name: NewUserSecret :exec
insert into user_secrets (user_id, salt, hash) values (sqlc.arg('user_id'), sqlc.arg('salt'), sqlc.arg('hash'));

-- name: NewAdminRole :exec
insert into roles (role) values (?)
on conflict(role) do nothing;

-- name: AdminExists :one
select count(*) > 0 as admin_exists
from user_roles
where role in (sqlc.slice('admin_roles'))
;

