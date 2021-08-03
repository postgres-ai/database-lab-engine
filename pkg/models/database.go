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
	DBName   string `json:"db_name"`
}
