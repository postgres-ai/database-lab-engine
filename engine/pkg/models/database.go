/*
2019 © Postgres.ai
*/

package models

// Database defines clone database parameters.
type Database struct {
	ConnStr  string `json:"connStr"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DBName   string `json:"dbName"`
	// OwnerUser is the authenticated user the clone is bound to (the full email
	// address). It is set only from the authenticated identity when clone binding
	// is enabled and is used as the trusted Teleport dblab_user label; it is
	// empty when binding is off or the creator used the shared token.
	OwnerUser string `json:"ownerUser,omitempty"`
}
