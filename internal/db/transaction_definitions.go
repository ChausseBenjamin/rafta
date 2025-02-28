package db

const RespMsgKey = "database_response"

type transactionName int

// XXX: Make sure iota length is always the same as commonTransactions
const (
	CreateUser transactionName = iota
	CreateUserSecret
	CreateTag
	CreateRole
	DeleteUser
	DeleteRole
	DeleteUnusedTags
	DeleteRoleFromUser
	DeleteTagFromTask
	GetSingleUser
	GetAllUsers
	GetSingleTask
	GetAllTasks
	GetSingleUserWithSecret
	GetAllTagsRelatedToTask
	AssignRoleToUser
	AssignTagToTask
	UpdateSetting
	GetUserRoles
	GetUserCount
)

var commonTransactions = [...]struct {
	Name transactionName
	Cmd  string
}{
	{ // Create a user (including salted secret)
		Name: CreateUser,
		Cmd:  "INSERT INTO Users (userID, name, email) VALUES (?, ?, ?)",
	},
	{ // Create user secrets
		Name: CreateUserSecret,
		Cmd:  "INSERT INTO UserSecrets (userID, saltAndHash) VALUES (?, ?)",
	},
	{ // Create a tag
		Name: CreateTag,
		Cmd:  "INSERT INTO Tags (name) VALUES (?)",
	},
	{ // Create a role
		Name: CreateRole,
		Cmd:  "INSERT INTO Roles (role) VALUES (?)",
	},
	{ // Remove a user
		Name: DeleteUser,
		Cmd:  "DELETE FROM Users WHERE userID = ?",
	},
	{ // Remove a role
		Name: DeleteRole,
		Cmd:  "DELETE FROM Roles WHERE role = ?",
	},
	{ // Remove unused tags (assigned to no tasks)
		Name: DeleteUnusedTags,
		Cmd:  "DELETE FROM Tags WHERE tagID NOT IN (SELECT tagID FROM TaskTags)",
	},
	{ // Assign a new role to a user
		Name: AssignRoleToUser,
		Cmd:  "INSERT INTO UserRoles (userID, role) VALUES (?, ?)",
	},
	{ // Assign a new tag to a task
		Name: AssignTagToTask,
		Cmd:  "INSERT INTO TaskTags (taskID, tagID) VALUES (?, ?)",
	},
	{ // Remove a role from a user
		Name: DeleteRoleFromUser,
		Cmd:  "DELETE FROM UserRoles WHERE userID = ? AND role = ?",
	},
	{ // Remove a tag from a task
		Name: DeleteTagFromTask,
		Cmd:  "DELETE FROM TaskTags WHERE taskID = ? AND tagID = ?",
	},
	{ // Update a setting KeyPair
		Name: UpdateSetting,
		Cmd:  "UPDATE Settings SET value = ? WHERE key = ?",
	},
	{ // Get a single user
		Name: GetSingleUser,
		Cmd:  "SELECT * FROM Users WHERE userID = ?",
	},
	{ // Get all users
		Name: GetAllUsers,
		Cmd:  "SELECT * FROM Users",
	},
	{ // Get a single task
		Name: GetSingleTask,
		Cmd:  "SELECT * FROM Tasks WHERE taskID = ?",
	},
	{ // Get all tasks
		Name: GetAllTasks,
		Cmd:  "SELECT * FROM Tasks",
	},
	{ // Get a single user with secret info and roles
		Name: GetSingleUserWithSecret,
		Cmd: `SELECT
			Users.name,
			Users.userID,
			Users.createdAt,
			Users.updatedAt,
			UserSecrets.saltAndHash
		FROM
			Users
		JOIN
				UserSecrets ON Users.userID = UserSecrets.userID
		WHERE
			Users.email = ?`,
	},
	{ // Get all tags related to a task
		Name: GetAllTagsRelatedToTask,
		Cmd: `SELECT t.* FROM Tags t
				JOIN TaskTags tt ON t.tagID = tt.tagID
				WHERE tt.taskID = ?`,
	},
	{ // Get all the roles assigned to a user
		Name: GetUserRoles,
		Cmd:  `SELECT role FROM UserRoles WHERE userID = ?`,
	},
	{ // Get how many users are signed up
		Name: GetUserCount,
		Cmd:  `SELECT COUNT(*) FROM Users`,
	},
}
