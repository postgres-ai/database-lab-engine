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
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"gitlab.com/postgres-ai/database-lab/v3/internal/cloning"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	dle_types "gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/networks"
)

const schemaDiffImage = "supabase/pgadmin-schema-diff"

// Diff defines a schema generator.
type Diff struct {
	cli *client.Client
	cl  *cloning.Base
	pm  *pool.Manager
}

// NewDiff creates a new Diff service.
func NewDiff(cli *client.Client, cl *cloning.Base, pm *pool.Manager) *Diff {
	return &Diff{cli: cli, cl: cl, pm: pm}
}

// GenerateDiff generate difference between database schemas.
func (d *Diff) GenerateDiff(ctx context.Context, actual *models.Clone, instanceID string) (string, error) {
	origin, err := d.createOriginClone(ctx, actual)
	if err != nil {
		return "", fmt.Errorf("cannot create a clone based on snapshot %s: %w", actual.Snapshot.ID, err)
	}

	defer func() {
		if err := d.cl.DestroyClone(origin.ID); err != nil {
			log.Err("Failed to destroy origin clone:", err)
		}
	}()

	diffContID, err := d.startDiffContainer(ctx, actual, origin, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to start diff container: %w", err)
	}

	defer func() {
		if err := d.cli.ContainerRemove(ctx, diffContID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
			log.Err("failed to remove the diff container:", diffContID, err)
		}
	}()

	statusCh, errCh := d.cli.ContainerWait(ctx, diffContID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("error on container waiting: %w", err)
		}
	case <-statusCh:
	}

	out, err := d.cli.ContainerLogs(ctx, diffContID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}

	buf := bytes.NewBuffer([]byte{})

	if _, err = stdcopy.StdCopy(buf, os.Stderr, out); err != nil {
		return "", fmt.Errorf("failed to copy container output: %w", err)
	}

	filteredOutput, err := filterOutput(buf)
	if err != nil {
		return "", fmt.Errorf("failed to filter output: %w", err)
	}

	return filteredOutput.String(), nil
}

// startDiffContainer starts a new diff container.
func (d *Diff) startDiffContainer(ctx context.Context, actual, origin *models.Clone, instanceID string) (string, error) {
	if err := tools.PullImage(ctx, d.cli, schemaDiffImage); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	fsm, err := d.pm.GetFSManager(actual.Snapshot.Pool)
	if err != nil {
		return "", fmt.Errorf("failed to get pool filesystem manager %s: %w", actual.Snapshot.ID, err)
	}

	originSocketDir := fsm.Pool().SocketCloneDir(util.GetCloneNameStr(origin.DB.Port))
	actualSocketDir := fsm.Pool().SocketCloneDir(util.GetCloneNameStr(actual.DB.Port))

	diffCont, err := d.cli.ContainerCreate(ctx,
		&container.Config{
			Labels: map[string]string{
				cont.DBLabControlLabel:    cont.DBLabSchemaDiff,
				cont.DBLabInstanceIDLabel: instanceID,
			},
			Image: schemaDiffImage,
			Cmd: []string{
				connString(actual, actualSocketDir),
				connString(origin, originSocketDir),
			},
		},
		&container.HostConfig{
			Mounts: d.getDiffMounts(actualSocketDir, originSocketDir),
		},
		&network.NetworkingConfig{},
		nil,
		d.cloneDiffName(actual),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create diff container: %w", err)
	}

	if err := networks.Connect(ctx, d.cli, instanceID, diffCont.ID); err != nil {
		return "", fmt.Errorf("failed to connect UI container to the internal Docker network: %w", err)
	}

	if err = d.cli.ContainerStart(ctx, diffCont.ID, types.ContainerStartOptions{}); err != nil {
		return "", fmt.Errorf("failed to create diff container: %w", err)
	}

	return diffCont.ID, nil
}

func (d *Diff) createOriginClone(ctx context.Context, clone *models.Clone) (*models.Clone, error) {
	originClone, err := d.cl.CreateClone(&dle_types.CloneCreateRequest{
		ID: d.cloneDiffName(clone),
		DB: &dle_types.DatabaseRequest{
			Username: clone.DB.Username,
			DBName:   clone.DB.DBName,
		},
		Snapshot: &dle_types.SnapshotCloneFieldRequest{ID: clone.Snapshot.ID},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create a clone based on snapshot %s: %w", clone.Snapshot.ID, err)
	}

	if originClone.Status.Code != models.StatusOK {
		if _, err := d.watchCloneStatus(ctx, originClone, originClone.Status.Code); err != nil {
			return nil, fmt.Errorf("failed to watch the clone status: %w", err)
		}
	}

	return originClone, nil
}

func (d *Diff) getDiffMounts(actualHost, originHost string) []mount.Mount {
	return []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: actualHost,
			Target: actualHost,
		},
		{
			Type:   mount.TypeBind,
			Source: originHost,
			Target: originHost,
		},
	}
}

func (d *Diff) cloneDiffName(actual *models.Clone) string {
	return "clone-diff-" + actual.ID
}

func connString(clone *models.Clone, socketDir string) string {
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s",
		socketDir, clone.DB.Port, clone.DB.Username, clone.DB.DBName)
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
	filteredBuilder := &strings.Builder{}

	for {
		line, err := b.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return filteredBuilder, nil
			}

			return nil, err
		}

		// Filter empty lines, comments and warnings.
		if len(line) == 0 || bytes.HasPrefix(line, []byte("--")) || bytes.HasPrefix(line, []byte("NOTE:")) {
			continue
		}

		filteredBuilder.Write(line)
	}
}
