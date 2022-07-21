package source

import (
	"context"
	"fmt"
	"os"

	"github.com/AlekSi/pointer"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	gitlabType = "gitlab"
	zipFormat  = "zip"
)

// GLProvider declares GitLab code provider.
type GLProvider struct {
	client *gitlab.Client
}

// NewGLProvider create a new GitLab code provider.
func NewGLProvider(token string) (*GLProvider, error) {
	git, err := gitlab.NewClient(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return &GLProvider{
		client: git,
	}, nil
}

// Download downloads the source code repository.
func (gl *GLProvider) Download(_ context.Context, opts Opts, outputFile string) error {
	log.Dbg(fmt.Sprintf("Download options: %#v", opts))

	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}

	defer func() { _ = f.Close() }()

	_, err = gl.client.Repositories.StreamArchive(opts.Repo, f, &gitlab.ArchiveOptions{
		Format: pointer.ToString(zipFormat),
		SHA:    &opts.Ref,
	})

	if err != nil {
		return fmt.Errorf("failed to download repository: %w", err)
	}

	return nil
}
