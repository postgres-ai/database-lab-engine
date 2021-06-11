/*
2021 © Postgres.ai
*/

// Package source provides a tools to use version control systems.
package source

import (
	"context"
)

const (
	// RepoDir defines a directory to clone and extract repository.
	RepoDir = "/tmp"
)

// Config describes the configuration of the plugged version control system.
type Config struct {
	Type  string `yaml:"type"`
	Token string `yaml:"token"`
}

// Provider declares code provider interface.
type Provider interface {
	Download(ctx context.Context, opts Opts, output string) error
	Extract(file string) (sourceCodeDir string, err error)
}

// Opts declares repository options.
type Opts struct {
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	Ref         string `json:"ref"`
	Branch      string `json:"branch"`
	BranchLink  string `json:"branch_link"`
	Commit      string `json:"commit"`
	CommitLink  string `json:"commit_link"`
	RequestLink string `json:"request_link"`
	DiffLink    string `json:"diff_link"`
}
