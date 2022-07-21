/*
2021 © Postgres.ai
*/

// Package source provides a tools to use version control systems.
package source

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/go-github/v34/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const githubType = "github"

// GHProvider declares GitHub code provider.
type GHProvider struct {
	client *github.Client
}

// NewGHProvider create a new GitHub code provider.
func NewGHProvider(ctx context.Context, token string) *GHProvider {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
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
		github.Zipball, &github.RepositoryContentGetOptions{Ref: getRunRef(opts)}, true)
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

func getRunRef(opts Opts) string {
	ref := opts.Commit

	if ref == "" {
		ref = opts.Ref
	}

	return ref
}
