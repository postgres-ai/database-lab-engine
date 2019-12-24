/*
2019 © Postgres.ai
*/

package models

type Database struct {
	ConnStr  string `json:"connStr"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}
