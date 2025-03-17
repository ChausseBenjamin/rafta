package db

const RespMsgKey = "database_response"

type statementName int

// XXX: Make sure iota length is always the same as commonTransactions
// WILL lead to an "index out of range" otherwise!
const (
	AssertTaskExists statementName = iota
	AssertUserExists
	CreateTask
	CreateUser
	CreateUserSecret
	DeleteUser
	GetAllUsers
	GetUser
	GetUserCount
	GetUserIDFromEmail
	GetUserRoles
	GetUserWithSecret
	RevokeToken
	UpdateUser
	UpdateUserPasswd

// CreateTag
// CreateRole
// DeleteRole
// DeleteUnusedTags
// DeleteRoleFromUser
// DeleteTagFromTask
// GetSingleTask
// GetAllTasks
// GetAllTagsRelatedToTask
// AssignRoleToUser
// AssignTagToTask
)

var commonStatements = [...]struct {
	Name statementName
	Cmd  string
}{
	{
		Name: AssertTaskExists,
		Cmd:  `SELECT EXISTS(SELECT 1 FROM Tasks WHERE TaskID = ?)`,
	},
	{
		Name: AssertUserExists,
		Cmd:  `SELECT EXISTS(SELECT 1 FROM Users WHERE userID = ?)`,
	},
	{
		Name: CreateTask,
		Cmd: `INSERT INTO Tasks
					(taskID, title, priority, description due, do, cron, cronIsEnabled, owner)
					VALUES
					(?, ?, ?, ? ?, ?, ?, ?, ?)`,
	},
	{ // Create a user (including salted secret)
		Name: CreateUser,
		Cmd:  `INSERT INTO Users (userID, name, email) VALUES (?, ?, ?)`,
	},
	{ // Create user secrets
		Name: CreateUserSecret,
		Cmd:  `INSERT INTO UserSecrets (userID, saltAndHash) VALUES (?, ?)`,
	},
	{ // Remove a user
		Name: DeleteUser,
		Cmd:  `DELETE FROM Users WHERE userID = ?`,
	},
	{
		Name: GetAllUsers,
		Cmd:  `SELECT userID, name, email, createdAt, updatedAt FROM Users`,
	},
	{
		Name: GetUser,
		Cmd:  `SELECT name, email, createdAt, updatedAt FROM Users WHERE userID=(?)`,
	},
	{ // Get how many users are signed up
		Name: GetUserCount,
		Cmd:  `SELECT COUNT(*) FROM Users`,
	},
	{ // Get how many users are signed up
		Name: GetUserIDFromEmail,
		Cmd:  `SELECT userID FROM Users WHERE email=(?)`,
	},
	{ // Get all the roles assigned to a user
		Name: GetUserRoles,
		Cmd:  `SELECT role FROM UserRoles WHERE userID = ?`,
	},
	{ // Get a single user with secret info and roles
		Name: GetUserWithSecret,
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
	{ // Add a jwt token to the list of revoked tokens
		Name: RevokeToken,
		Cmd:  `INSERT INTO RevokedTokens (tokenID, expiration) VALUES (?, ?)`,
	},
	{
		Name: UpdateUser,
		Cmd: `UPDATE Users SET
						name = ?,
						email = ?,
						updatedAt = ?
					WHERE UserID = ?`,
	},
	{
		Name: UpdateUserPasswd,
		Cmd:  `Update UserSecrets SET saltAndHash = ? WHERE userID = ?`,
	},
	// { // Create a tag
	// 	Name: CreateTag,
	// 	Cmd:  "INSERT INTO Tags (name) VALUES (?)",
	// },
	// { // Create a role
	// 	Name: CreateRole,
	// 	Cmd:  "INSERT INTO Roles (role) VALUES (?)",
	// },
	// { // Remove a role
	// 	Name: DeleteRole,
	// 	Cmd:  "DELETE FROM Roles WHERE role = ?",
	// },
	// { // Remove unused tags (assigned to no tasks)
	// 	Name: DeleteUnusedTags,
	// 	Cmd:  "DELETE FROM Tags WHERE tagID NOT IN (SELECT tagID FROM TaskTags)",
	// },
	// { // Assign a new role to a user
	// 	Name: AssignRoleToUser,
	// 	Cmd:  "INSERT INTO UserRoles (userID, role) VALUES (?, ?)",
	// },
	// { // Assign a new tag to a task
	// 	Name: AssignTagToTask,
	// 	Cmd:  "INSERT INTO TaskTags (taskID, tagID) VALUES (?, ?)",
	// },
	// { // Remove a role from a user
	// 	Name: DeleteRoleFromUser,
	// 	Cmd:  "DELETE FROM UserRoles WHERE userID = ? AND role = ?",
	// },
	// { // Remove a tag from a task
	// 	Name: DeleteTagFromTask,
	// 	Cmd:  "DELETE FROM TaskTags WHERE taskID = ? AND tagID = ?",
	// },
	// { // Get a single user
	// 	Name: GetSingleUser,
	// 	Cmd:  "SELECT * FROM Users WHERE userID = ?",
	// },
	// { // Get all users
	// 	Name: GetAllUsers,
	// 	Cmd:  "SELECT * FROM Users",
	// },
	// { // Get a single task
	// 	Name: GetSingleTask,
	// 	Cmd:  "SELECT * FROM Tasks WHERE taskID = ?",
	// },
	// { // Get all tasks
	// 	Name: GetAllTasks,
	// 	Cmd:  "SELECT * FROM Tasks",
	// },
	// { // Get all tags related to a task
	// 	Name: GetAllTagsRelatedToTask,
	// 	Cmd: `SELECT t.* FROM Tags t
	// 			JOIN TaskTags tt ON t.tagID = tt.tagID
	// 			WHERE tt.taskID = ?`,
	// },
}
