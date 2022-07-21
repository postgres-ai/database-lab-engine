/*
2021 © Postgres.ai
*/

// Package source provides a tools to use version control systems.
package source

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	// RepoDir defines a directory to clone and extract repository.
	RepoDir = "/tmp/ci_checker"
)

// Config describes the configuration of the plugged version control system.
type Config struct {
	Type  string `yaml:"type"`
	Token string `yaml:"token"`
}

// Provider declares code provider interface.
type Provider interface {
	Download(ctx context.Context, opts Opts, output string) error
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

// NewCodeProvider creates a new code provider.
func NewCodeProvider(ctx context.Context, cfg *Config) (Provider, error) {
	switch cfg.Type {
	case gitlabType:
		return NewGLProvider(cfg.Token)

	case githubType:
		return NewGHProvider(ctx, cfg.Token), nil

	default:
		return nil, fmt.Errorf("code provider %q is not supported", cfg.Type)
	}
}

// ExtractArchive extracts the downloaded repository archive.
func ExtractArchive(file string) (string, error) {
	extractDirNameCmd := fmt.Sprintf("unzip -qql %s | head -n1 | tr -s ' ' | cut -d' ' -f5-", file)

	log.Dbg("Command: ", extractDirNameCmd)

	dirName, err := exec.Command("bash", "-c", extractDirNameCmd).Output()
	if err != nil {
		return "", err
	}

	log.Dbg("Archive directory: ", string(bytes.TrimSpace(dirName)))

	archiveDir, err := os.MkdirTemp(RepoDir, "*_extract")
	if err != nil {
		return "", err
	}

	resp, err := exec.Command("unzip", "-d", archiveDir, file).CombinedOutput()
	log.Dbg("Response: ", string(resp))

	if err != nil {
		return "", err
	}

	source := path.Join(archiveDir, string(bytes.TrimSpace(dirName)))
	log.Dbg("Source: ", source)

	return source, nil
}
