/*
2019 © Postgres.ai
*/

package models

type Clone struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Snapshot    *Snapshot `json:"snapshot"`
	CloneSize   uint64    `json:"cloneSize"`
	CloningTime float64   `json:"cloningTime"`
	Protected   bool      `json:"protected"`
	DeleteAt    string    `json:"deleteAt"`
	CreatedAt   string    `json:"createdAt"`
	Status      *Status   `json:"status"`
	Db          *Database `json:"db"`

	// TODO(anatoly): Remove?
	Project string `json:"project"`
}
