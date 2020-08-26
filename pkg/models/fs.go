/*
2019 © Postgres.ai
*/

package models

type FileSystem struct {
	Size uint64 `json:"size"`
	Free uint64 `json:"free"`
	Used uint64 `json:"used"`
}
