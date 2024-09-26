/*
2022 © Postgres.ai
*/

package db

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/jackc/pgx/v4"

	dockerTools "gitlab.com/postgres-ai/database-lab/v3/internal/provision/docker"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/health"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/networks"
)

const (
	extensionQuery = "select jsonb_object_agg(name, default_version) from pg_available_extensions"

	port     = "5432"
	username = "postgres"
	dbname   = "postgres"
	password = ""

	foundationName = "dblab_foundation_"

	defaultRetries = 10
)

// ImageContent keeps the content lists from the foundation image.
type ImageContent struct {
	engineProps global.EngineProps
	isReady     bool
	extensions  map[string]string
	locales     map[string]struct{}
	databases   map[string]struct{}
}

// IsReady reports if the ImageContent has collected details about the current image.
func (i *ImageContent) IsReady() bool {
	return i.isReady
}

// NewImageContent creates a new ImageContent.
func NewImageContent(engineProps global.EngineProps) *ImageContent {
	return &ImageContent{
		engineProps: engineProps,
		extensions:  make(map[string]string, 0),
		locales:     make(map[string]struct{}, 0),
		databases:   make(map[string]struct{}, 0),
	}
}

// Extensions provides list of Postgres extensions from the foundation image.
func (i *ImageContent) Extensions() map[string]string {
	return i.extensions
}

// Locales provides list of locales from the foundation image.
func (i *ImageContent) Locales() map[string]struct{} {
	return i.locales
}

// SetDatabases sets a list of databases mentioned in the Retrieval config.
// An empty list means all databases.
func (i *ImageContent) SetDatabases(dbList []string) {
	if len(dbList) == 0 {
		i.databases = make(map[string]struct{}, 0)
		return
	}

	for _, dbName := range dbList {
		i.databases[dbName] = struct{}{}
	}
}

// Databases returns the list of databases mentioned in the Retrieval config.
// An empty list means all databases.
func (i *ImageContent) Databases() map[string]struct{} {
	return i.databases
}

// Collect collects extension and locale lists from the provided Docker image.
func (i *ImageContent) Collect(dockerImage string) error {
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.39"))
	if err != nil {
		log.Fatal("Failed to create a Docker client:", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	if err := i.collectImageContent(ctx, docker, dockerImage); err != nil {
		return err
	}

	i.isReady = true

	log.Msg("The image content has been successfully collected")

	return nil
}

func getFoundationName(instanceID string) string {
	return foundationName + instanceID
}

func (i *ImageContent) collectImageContent(ctx context.Context, docker *client.Client, dockerImage string) error {
	containerID, err := createContainer(ctx, docker, dockerImage, i.engineProps)
	if err != nil {
		return fmt.Errorf("failed to create a Docker container: %w", err)
	}

	defer tools.RemoveContainer(ctx, docker, containerID, 0)

	if err := i.collectExtensions(ctx, i.engineProps.InstanceID); err != nil {
		return fmt.Errorf("failed to collect extensions from the image %s: %w", dockerImage, err)
	}

	if err := i.collectLocales(ctx, docker, containerID); err != nil {
		return fmt.Errorf("failed to collect locales: %w", err)
	}

	return nil
}

func (i *ImageContent) collectExtensions(ctx context.Context, instanceID string) error {
	conn, err := pgx.Connect(ctx, ConnectionString(getFoundationName(instanceID), port, username, dbname, password))
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	var row []byte

	if err = conn.QueryRow(ctx, extensionQuery).Scan(&row); err != nil {
		return err
	}

	extensionMap := map[string]string{}

	if err := json.Unmarshal(row, &extensionMap); err != nil {
		return err
	}

	i.extensions = extensionMap

	return nil
}

func (i *ImageContent) collectLocales(ctx context.Context, docker *client.Client, containerID string) error {
	out, err := getLocales(ctx, docker, containerID)
	if err != nil {
		return err
	}

	imageLocales := map[string]struct{}{}

	for _, line := range strings.Split(out, "\n") {
		if len(line) != 0 {
			locale := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(line)), "-", "")
			imageLocales[locale] = struct{}{}
		}
	}

	i.locales = imageLocales

	return nil
}

func createContainer(ctx context.Context, docker *client.Client, image string, props global.EngineProps) (string, error) {
	if err := dockerTools.PrepareImage(ctx, docker, image); err != nil {
		return "", fmt.Errorf("failed to prepare Docker image: %w", err)
	}

	containerConf := &container.Config{
		Labels: map[string]string{
			cont.DBLabControlLabel:    cont.DBLabFoundationLabel,
			cont.DBLabInstanceIDLabel: props.InstanceID,
			cont.DBLabEngineNameLabel: props.ContainerName,
		},
		Env: []string{
			"POSTGRES_HOST_AUTH_METHOD=trust",
		},
		Image: image,
		Healthcheck: health.GetConfig(username, dbname,
			health.OptionInterval(health.DefaultRestoreInterval), health.OptionRetries(defaultRetries)),
	}

	containerName := getFoundationName(props.InstanceID)

	containerID, err := tools.CreateContainerIfMissing(ctx, docker, containerName, containerConf, &container.HostConfig{})
	if err != nil {
		return "", fmt.Errorf("failed to create container %q %w", containerName, err)
	}

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", containerName, containerID))

	if err := docker.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container %q: %w", containerName, err)
	}

	if err := tools.InitDB(ctx, docker, containerID); err != nil {
		return "", fmt.Errorf("failed to init Postgres: %w", err)
	}

	if err := resetHBA(ctx, docker, containerID); err != nil {
		return "", fmt.Errorf("failed to prepare pg_hba.conf: %w", err)
	}

	if err := setListenAddresses(ctx, docker, containerID); err != nil {
		return "", fmt.Errorf("failed to set listen_addresses: %w", err)
	}

	if err := tools.StartPostgres(ctx, docker, containerID, tools.DefaultStopTimeout); err != nil {
		return "", fmt.Errorf("failed to init Postgres: %w", err)
	}

	log.Dbg("Waiting for container readiness")

	if err := tools.CheckContainerReadiness(ctx, docker, containerID); err != nil {
		return "", fmt.Errorf("failed to readiness check: %w", err)
	}

	if err := networks.Connect(ctx, docker, props.InstanceID, containerID); err != nil {
		return "", fmt.Errorf("failed to connect UI container to the internal Docker network: %w", err)
	}

	return containerID, nil
}

func resetHBA(ctx context.Context, dockerClient *client.Client, containerID string) error {
	command := []string{"sh", "-c", `su postgres -c "echo 'hostnossl all all 0.0.0.0/0 trust' > ${PGDATA}/pg_hba.conf"`}

	log.Dbg("Reset pg_hba", command)

	out, err := tools.ExecCommandWithOutput(ctx, dockerClient, containerID, types.ExecConfig{
		Tty: true,
		Cmd: command,
	})

	if err != nil {
		log.Dbg(out)
		return fmt.Errorf("failed to reset pg_hba.conf: %w", err)
	}

	return nil
}

func setListenAddresses(ctx context.Context, dockerClient *client.Client, containerID string) error {
	command := []string{"sh", "-c", `su postgres -c "echo listen_addresses = \'*\' >> ${PGDATA}/postgresql.conf"`}

	log.Dbg("Set listen addresses", command)

	out, err := tools.ExecCommandWithOutput(ctx, dockerClient, containerID, types.ExecConfig{
		Tty: true,
		Cmd: command,
	})

	if err != nil {
		log.Dbg(out)
		return fmt.Errorf("failed to set listen addresses: %w", err)
	}

	return nil
}

func getLocales(ctx context.Context, dockerClient *client.Client, containerID string) (string, error) {
	command := []string{"sh", "-c", `locale -a`}

	log.Dbg("Get locale list", command)

	out, err := tools.ExecCommandWithOutput(ctx, dockerClient, containerID, types.ExecConfig{
		Tty: true,
		Cmd: command,
	})

	if err != nil {
		return "", fmt.Errorf("failed to get locale list: %w", err)
	}

	return out, nil
}
