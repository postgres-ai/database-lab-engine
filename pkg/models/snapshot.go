/*
2019 © Postgres.ai
*/

package models

type Snapshot struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"createdAt"`
	DataStateAt string `json:"dataStateAt"`
}
