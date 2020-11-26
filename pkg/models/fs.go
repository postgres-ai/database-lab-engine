/*
2019 © Postgres.ai
*/

package models

// FileSystem describes state of a file system.
type FileSystem struct {
	Size   uint64 `json:"size"`
	SizeHR string `json:"sizeHR"`
	Free   uint64 `json:"free"`
	FreeHR string `json:"freeHR"`
	Used   uint64 `json:"used"`
	UsedHR string `json:"usedHR"`
}
