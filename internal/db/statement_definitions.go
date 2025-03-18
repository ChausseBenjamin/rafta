package db

const RespMsgKey = "database_response"

type statementName int

// This is the index of a prepared statement in the store
// therefore ensuring an O(1) operation to access any prepared
// query in the server.
const (
	AssertTaskExists statementName = iota
	AssertUserExists
	CreateTag
	CreateTask
	CreateTaskTag
	CreateUser
	CreateUserSecret
	DeleteTaskTag
	DeleteUser
	GetAllUsers
	GetTagID
	GetTaskTags
	GetUser
	GetUserCount
	GetUserIDFromEmail
	GetUserRoles
	GetUserTask
	GetUserTasks
	GetUserWithSecret
	RevokeToken
	UpdateUser
	UpdateUserPasswd
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
		Name: CreateTag,
		Cmd:  `INSERT OR IGNORE INTO Tags (name) VALUES (?)`,
	},
	{
		Name: CreateTask,
		Cmd: `INSERT INTO Tasks (
						taskID,
						title,
						priority,
						description,
						due,
						do,
						recurrencePattern,
						recurrenceEnabled,
						createdAt,
						updatedAt,
						owner)
					VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	},
	{
		Name: CreateTaskTag,
		Cmd:  `INSERT INTO TaskTags (taskID, tagID) VALUES (?, ?)`,
	},
	{
		Name: CreateUser,
		Cmd:  `INSERT INTO Users (userID, name, email) VALUES (?, ?, ?)`,
	},
	{
		Name: CreateUserSecret,
		Cmd:  `INSERT INTO UserSecrets (userID, saltAndHash) VALUES (?, ?)`,
	},
	{
		Name: DeleteTaskTag,
		Cmd:  `DELETE FROM TaskTags WHERE taskID = ? AND tagID = ?`,
	},
	{
		Name: DeleteUser,
		Cmd:  `DELETE FROM Users WHERE userID = ?`,
	},
	{
		Name: GetAllUsers,
		Cmd:  `SELECT userID, name, email, createdAt, updatedAt FROM Users`,
	},
	{
		Name: GetTagID,
		Cmd:  `SELECT tagID FROM Tags WHERE name = ?`,
	},
	{
		Name: GetTaskTags,
		Cmd: `SELECT t.name, t.tagID
			FROM Tags t
			INNER JOIN TaskTags tt ON t.tagID = tt.tagID
			WHERE tt.taskID = ?`,
	},
	{
		Name: GetUser,
		Cmd:  `SELECT name, email, createdAt, updatedAt FROM Users WHERE userID=(?)`,
	},
	{
		Name: GetUserCount,
		Cmd:  `SELECT COUNT(*) FROM Users`,
	},
	{
		Name: GetUserIDFromEmail,
		Cmd:  `SELECT userID FROM Users WHERE email= ?`,
	},
	{
		Name: GetUserRoles,
		Cmd:  `SELECT role FROM UserRoles WHERE userID = ?`,
	},
	{
		Name: GetUserTask,
		Cmd: `SELECT
						title,
						priority,
						description,
						due,
						do,
						recurrencePattern,
						recurrenceEnabled,
						createdAt,
						updatedAt
					FROM Tasks WHERE owner= ? AND taskID = ?`,
	},
	{
		Name: GetUserTasks,
		Cmd: `SELECT
						taskID,
						title,
						priority,
						description,
						due,
						do,
						recurrencePattern,
						recurrenceEnabled,
						createdAt,
						updatedAt
					FROM Tasks WHERE owner= ?`,
	},
	{
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
	{
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
}
