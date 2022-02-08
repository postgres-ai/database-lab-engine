/*
2021 © Postgres.ai
*/

// Package source provides a tools to use version control systems.
package source

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"

	"github.com/google/go-github/v34/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

// GHProvider declares GitHub code provider.
type GHProvider struct {
	client *github.Client
}

// NewCodeProvider create a new code provider
func NewCodeProvider(ctx context.Context, cfg *Config) *GHProvider {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.Token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &GHProvider{
		client: github.NewClient(tc),
	}
}

// Download downloads repository.
func (cp *GHProvider) Download(ctx context.Context, opts Opts, outputFile string) error {
	log.Dbg(fmt.Sprintf("Download options: %#v", opts))

	archiveLink, _, err := cp.client.Repositories.GetArchiveLink(ctx, opts.Owner, opts.Repo,
		github.Zipball, &github.RepositoryContentGetOptions{Ref: opts.Ref}, true)
	if err != nil {
		return errors.Wrap(err, "failed to download content")
	}

	log.Dbg("Archive link", archiveLink.String())

	archiveResponse, err := http.Get(archiveLink.String())
	if err != nil {
		return errors.Wrap(err, "failed to get content")
	}

	defer func() { _ = archiveResponse.Body.Close() }()

	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}

	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, archiveResponse.Body); err != nil {
		return err
	}

	return nil
}

// Extract extracts downloaded repository archive.
func (cp *GHProvider) Extract(file string) (string, error) {
	extractDirNameCmd := fmt.Sprintf("unzip -qql %s | head -n1 | tr -s ' ' | cut -d' ' -f5-", file)

	log.Dbg("Command: ", extractDirNameCmd)

	dirName, err := exec.Command("bash", "-c", extractDirNameCmd).Output()
	if err != nil {
		return "", err
	}

	log.Dbg("Archive directory: ", string(bytes.TrimSpace(dirName)))

	resp, err := exec.Command("unzip", "-d", RepoDir, file).CombinedOutput()
	log.Dbg("Response: ", string(resp))

	if err != nil {
		return "", err
	}

	source := path.Join(RepoDir, string(bytes.TrimSpace(dirName)))
	log.Dbg("Source: ", source)

	return source, nil
}
