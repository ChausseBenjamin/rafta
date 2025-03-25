-- name: NewTag :one
insert into tags (name) values (?) returning *;

-- name: GetAllTags :many
select t.name
from tags t
;

-- name: GetExistingTags :many
select *
from tags
where name in (sqlc.slice('tags'))
;


-- name: GetTaskTags :many
select tags.*
from tags
inner join task_tags tt on tags.tag_id = tt.tag_id
where tt.task_id = ?
;

-- name: AssignTag :exec
insert into task_tags (task_id, tag_id) values (sqlc.arg('task'), sqlc.arg('tag'));

-- name: GetTaskTagsNames :many
select tags.name
from tags
inner join task_tags tt on tags.tag_id = tt.tag_id
where tt.task_id = ?
;

-- name: GetTagsWithNames :many
select *
from tags
where name in (sqlc.slice('tags'))
;

-- name: UnassignTags :exec
delete from task_tags
where tag_id in (sqlc.slice('tag_ids'))
;

-- name: CleanTags :exec
delete from tags
where tag_id not in (select distinct tag_id from task_tags)
;

