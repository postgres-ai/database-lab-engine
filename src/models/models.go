/*
2019 © Postgres.ai
*/

package models

type Status struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Database struct {
	ConnStr  string `json:"connStr"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Disk struct {
	Size uint64 `json:"size"`
	Free uint64 `json:"free"`
}

type Snapshot struct {
	Id        string `json:"id"`
	Timestamp string `json:"timestamp"`
}

type Clone struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Project     string    `json:"project"`
	Snapshot    string    `json:"snapshot"`
	CloneSize   uint64    `json:"cloneSize"`
	CloningTime uint64    `json:"cloningTime"`
	Protected   bool      `json:"protected"`
	DeleteAt    string    `json:"deleteAt"`
	CreatedAt   string    `json:"createdAt"`
	Status      *Status   `json:"status"`
	Db          *Database `json:"db"`
}

type InstanceStatus struct {
	Status              *Status  `json:"status"`
	Disk                *Disk    `json:"disk"`
	DataSize            uint64   `json:"dataSize"`
	ExpectedCloningTime float64  `json:"expectedCloningTime"`
	NumClones           uint64   `json:"numClones"`
	Clones              []*Clone `json:"clones"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
	Hint    string `json:"hint"`
}
