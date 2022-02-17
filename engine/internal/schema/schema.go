// Package schema provides tools to manage PostgreSQL schemas difference.
package schema

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/networks"
)

// Diff defines a schema generator.
type Diff struct {
	cli *client.Client
}

// NewDiff creates a new Diff service.
func NewDiff(cli *client.Client) *Diff {
	return &Diff{cli: cli}
}

func connStr(clone *models.Clone) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		clone.DB.Username,
		"test", // clone.DB.Password,
		util.GetCloneNameStr(clone.DB.Port),
		clone.DB.Port,
		"test", // clone.DB.DBName,
	)
}

// GenerateDiff generate difference between database schemas.
func (d *Diff) GenerateDiff(actual, origin *models.Clone, instanceID string) (string, error) {
	log.Dbg("Origin clone:", origin)
	log.Dbg("Actual clone:", actual.DB.ConnStr+" password=test")

	ctx := context.Background()

	if _, err := d.watchCloneStatus(ctx, origin, origin.Status.Code); err != nil {
		return "", fmt.Errorf("failed to watch the clone status: %w", err)
	}

	reader, err := d.cli.ImagePull(ctx, "supabase/pgadmin-schema-diff", types.ImagePullOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	defer func() { _ = reader.Close() }()

	if _, err := io.Copy(os.Stdout, reader); err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	diffCont, err := d.cli.ContainerCreate(ctx,
		&container.Config{
			Labels: map[string]string{
				cont.DBLabControlLabel:    cont.DBLabSchemaDiff,
				cont.DBLabInstanceIDLabel: instanceID,
				// cont.DBLabEngineNameLabel: d.engineProps.ContainerName,
			},
			Image: "supabase/pgadmin-schema-diff",
			Cmd: []string{
				connStr(actual),
				connStr(origin),
			},
		},
		&container.HostConfig{},
		&network.NetworkingConfig{},
		nil,
		"clone-diff-"+actual.ID,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create diff container: %w", err)
	}

	if err := networks.Connect(ctx, d.cli, instanceID, diffCont.ID); err != nil {
		return "", fmt.Errorf("failed to connect UI container to the internal Docker network: %w", err)
	}

	err = d.cli.ContainerStart(ctx, diffCont.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create diff container: %w", err)
	}

	statusCh, errCh := d.cli.ContainerWait(ctx, diffCont.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("error on container waiting: %w", err)
		}
	case <-statusCh:
	}

	out, err := d.cli.ContainerLogs(ctx, diffCont.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}

	buf := bytes.NewBuffer([]byte{})

	_, err = stdcopy.StdCopy(buf, os.Stderr, out)
	if err != nil {
		return "", fmt.Errorf("failed to copy container output: %w", err)
	}

	stringsB, err := filterOutput(buf)
	if err != nil {
		return "", fmt.Errorf("failed to filter output: %w", err)
	}

	return stringsB.String(), nil
}

// watchCloneStatus checks the clone status for changing.
func (d *Diff) watchCloneStatus(ctx context.Context, clone *models.Clone, initialStatusCode models.StatusCode) (*models.Clone, error) {
	const pollingInterval = 5 * time.Second

	pollingTimer := time.NewTimer(pollingInterval)
	defer pollingTimer.Stop()

	var cancel context.CancelFunc

	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, time.Minute)
		defer cancel()
	}

	for {
		select {
		case <-pollingTimer.C:
			log.Dbg("Check status:", clone.Status.Code)

			if clone.Status.Code != initialStatusCode {
				return clone, nil
			}

			pollingTimer.Reset(pollingInterval)

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func filterOutput(b *bytes.Buffer) (*strings.Builder, error) {
	strB := &strings.Builder{}

	for {
		line, err := b.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return strB, nil
			}

			return nil, err
		}

		if len(line) == 0 || bytes.HasPrefix(line, []byte("--")) || bytes.HasPrefix(line, []byte("NOTE:")) {
			continue
		}

		strB.Write(line)
	}
}
