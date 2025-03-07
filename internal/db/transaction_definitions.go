package db

const RespMsgKey = "database_response"

type transactionName int

// XXX: Make sure iota length is always the same as commonTransactions
const (
	CreateUser transactionName = iota
	CreateUserSecret
	DeleteUser
	GetSingleUserWithSecret
	GetUserCount
	GetUserRoles
	RevokeToken

// CreateTag
// CreateRole
// DeleteRole
// DeleteUnusedTags
// DeleteRoleFromUser
// DeleteTagFromTask
// GetSingleUser
// GetAllUsers
// GetSingleTask
// GetAllTasks
// GetAllTagsRelatedToTask
// AssignRoleToUser
// AssignTagToTask
// UpdateSetting
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
	{ // Remove a user
		Name: DeleteUser,
		Cmd:  "DELETE FROM Users WHERE userID = ?",
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
	{ // Get how many users are signed up
		Name: GetUserCount,
		Cmd:  `SELECT COUNT(*) FROM Users`,
	},
	{ // Get all the roles assigned to a user
		Name: GetUserRoles,
		Cmd:  `SELECT role FROM UserRoles WHERE userID = ?`,
	},
}
