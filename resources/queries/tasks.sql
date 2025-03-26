-- name: GetTask :one
select *
from tasks
where task_id = ?
;

-- name: GetUserTask :one
select *
from tasks
where task_id = ? and owner = ?
;

-- name: GetUserTasks :many
select *
from tasks
where owner = ?
;

-- name: NewTask :one
insert into tasks
(title, state, priority, description, due_date, do_date, recurrence_pattern, recurrence_enabled, owner) values
(?, ?, ?, ?, ?, ?, ?, ?, ?) returning *;

-- name: DeleteUserTask :execrows
delete from tasks
where owner = sqlc.arg('owner') and task_id = sqlc.arg('task')
;

