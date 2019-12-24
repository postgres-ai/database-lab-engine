/*
2019 © Postgres.ai
*/

package models

type Snapshot struct {
	Id          string `json:"id"`
	CreatedAt   string `json:"createdAt"`
	DataStateAt string `json:"dataStateAt"`
}
