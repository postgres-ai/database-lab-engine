/*
2021 © Postgres.ai
*/

package runci

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/pkg/errors"
	"github.com/rs/xid"

	"gitlab.com/postgres-ai/database-lab/v3/internal/runci/source"

	"gitlab.com/postgres-ai/database-lab/v3/internal/observer"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/cont"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"

	dblab_types "gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
	"gitlab.com/postgres-ai/database-lab/v3/version"
)

const (
	repoDirInRunner    = "/repo"
	outputFileTemplate = "repo_%s.zip"
)

// StartMigrationRequest defines a request to start migration check.
type StartMigrationRequest struct {
	Source            source.Opts        `json:"source"`
	Username          string             `json:"username"`
	UsernameFull      string             `json:"username_full"`
	UsernameLink      string             `json:"username_link"`
	DBName            string             `json:"db_name"`
	Commands          []string           `json:"commands"`
	MigrationEnvs     []string           `json:"migration_envs"`
	ObservationConfig dblab_types.Config `json:"observation_config"`
	KeepClone         bool               `json:"keep_clone"`
}

// MigrationResult provides the results of the executed migration.
type MigrationResult struct {
	CloneID string            `json:"clone_id"`
	Session *observer.Session `json:"session"`
}

// runMigration runs database migration.
func (s *Server) runMigration(w http.ResponseWriter, r *http.Request) {
	request := StartMigrationRequest{}

	if err := readJSON(r, &request); err != nil {
		log.Errf("failed to read request: %v", err)
		api.SendBadRequestError(w, r, err.Error())

		return
	}

	runID := xid.New().String()
	outputFile := path.Join(source.RepoDir, fmt.Sprintf(outputFileTemplate, runID))

	if err := s.codeProvider.Download(context.Background(), request.Source, outputFile); err != nil {
		log.Errf("failed to download: %v", err)
		api.SendBadRequestError(w, r, err.Error())

		return
	}

	defer func() {
		if err := os.Remove(outputFile); err != nil {
			log.Dbg("failed to remove file: ", err)
		}
	}()

	sourceCodeDir, err := s.codeProvider.Extract(outputFile)
	if err != nil {
		api.SendError(w, r, err)
		return
	}

	defer func() {
		if err := os.RemoveAll(sourceCodeDir); err != nil {
			log.Dbg("failed to remove the source code directory: ", err)
		}
	}()

	volumes := map[string]string{
		sourceCodeDir: repoDirInRunner,
	}

	log.Dbg(volumes)

	clone, err := createDBLabClone(context.Background(), s.dle, cloneOpts{
		username: request.Username,
		dbname:   request.DBName,
	})
	if err != nil {
		api.SendError(w, r, err)
		return
	}

	log.Dbg("Clone: ", clone)

	dleHealth, err := s.dle.Health(context.Background())
	if err != nil {
		api.SendError(w, r, err)
		return
	}

	tags := map[string]string{
		"launched_by":   request.Username,
		"username_full": request.UsernameFull,
		"username_link": request.UsernameLink,
		"revision":      request.Source.Commit,
		"revision_link": request.Source.CommitLink,
		"request_link":  request.Source.RequestLink,
		"branch":        request.Source.Branch,
		"branch_link":   request.Source.BranchLink,
		"diff_link":     request.Source.DiffLink,
		"data_state_at": clone.Snapshot.DataStateAt,
		"dle_version":   dleHealth.Version,
	}

	session, err := s.runCommands(context.Background(), clone, runID, volumes, tags, request.Commands, request.MigrationEnvs,
		request.ObservationConfig)
	if err != nil {
		api.SendError(w, r, err)
		return
	}

	if !request.KeepClone {
		if err := s.dle.DestroyClone(context.Background(), clone.ID); err != nil {
			log.Errf("failed to destroy clone: %v", err)
		}
	}

	migrationResult := MigrationResult{
		CloneID: clone.ID,
		Session: session,
	}

	migrationResponse, err := json.Marshal(migrationResult)
	if err != nil {
		api.SendError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(migrationResponse)
}

func (s *Server) runCommands(ctx context.Context, clone *models.Clone, runID string, volumes, tags map[string]string,
	commands, migrationEnvs []string, cfg dblab_types.Config) (*observer.Session, error) {
	if err := tools.PullImage(ctx, s.docker, s.config.Runner.Image); err != nil {
		return nil, errors.Wrap(err, "failed to scan pulling image response")
	}

	containerCfg := s.buildContainerConfig(clone, migrationEnvs)

	log.Dbg(containerCfg)

	mounts := make([]mount.Mount, 0, len(volumes))
	for sourcePath, targetPath := range volumes {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: sourcePath,
			Target: targetPath,
		})
	}

	hostConfig := &container.HostConfig{
		Mounts: mounts,
	}
	networkConfig := &network.NetworkingConfig{}
	containerName := "migration_runner_" + runID

	if s.networkID != "" {
		networkConfig.EndpointsConfig = map[string]*network.EndpointSettings{"clone_network": {NetworkID: s.networkID}}
	}

	contRunner, err := s.docker.ContainerCreate(ctx, containerCfg, hostConfig, networkConfig, nil, containerName)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create container")
	}

	defer tools.RemoveContainer(ctx, s.docker, contRunner.ID, cont.StopPhysicalTimeout)

	log.Dbg("ContainerID: ", contRunner.ID)

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", containerName, contRunner.ID))

	if err := s.docker.ContainerStart(ctx, contRunner.ID, types.ContainerStartOptions{}); err != nil {
		return nil, errors.Wrapf(err, "failed to start container %q", containerName)
	}

	session, err := s.dle.StartObservation(ctx,
		dblab_types.StartObservationRequest{
			CloneID: clone.ID,
			Tags:    tags,
			Config:  cfg,
		},
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to start observation session")
	}

	defer func() {
		if !session.IsFinished() {
			log.Msg("Session has not been finished properly. Stop observation")

			if _, stopErr := s.dle.StopObservation(ctx, dblab_types.StopObservationRequest{CloneID: clone.ID, OverallError: true}); stopErr != nil {
				log.Err(errors.Wrap(stopErr, "failed to stop observation session"))
			}
		}
	}()

	for _, command := range commands {
		cmd := []string{"/bin/sh", "-c", command}

		log.Msg("Running command: ", cmd)

		output, err := tools.ExecCommandWithOutput(ctx, s.docker, contRunner.ID, types.ExecConfig{
			Cmd: cmd,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to execute command")
		}

		log.Msg("Command output: ", output)

		log.Msg("Command has been executed: ", cmd)
	}

	session, err = s.dle.StopObservation(ctx, dblab_types.StopObservationRequest{CloneID: clone.ID})
	if err != nil {
		log.Err(errors.Wrap(err, "failed to stop observation session"))
	}

	log.Msg("Observation session: ", session.SessionID)

	sessionResponse, err := json.MarshalIndent(session, "", "    ")
	if err != nil {
		return nil, err
	}

	log.Msg("Observation session output: ", string(sessionResponse))

	return session, nil
}

func (s *Server) buildContainerConfig(clone *models.Clone, migrationEnvs []string) *container.Config {
	host := clone.DB.Host
	if host == s.dle.URL("").Hostname() || host == "127.0.0.1" || host == "localhost" {
		host = util.GetCloneNameStr(clone.DB.Port)
	}

	return &container.Config{
		Labels: map[string]string{
			cont.DBLabRunner: cont.DBLabRunner,
		},
		Image: s.config.Runner.Image,
		Env: append([]string{
			"PGUSER=" + clone.DB.Username,
			"PGPASSWORD=" + clone.DB.Password,
			"PGHOST=" + host,
			"PGPORT=" + clone.DB.Port,
			"PGDATABASE=" + clone.DB.DBName,
		}, migrationEnvs...),
	}
}

func (s *Server) downloadArtifact(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	artifactType := values.Get("artifact_type")

	if !observer.IsAvailableArtifactType(artifactType) {
		api.SendBadRequestError(w, r, fmt.Sprintf("artifact %q is not available to download", artifactType))
		return
	}

	sessionID := values.Get("session_id")
	cloneID := values.Get("clone_id")

	body, err := s.dle.DownloadArtifact(r.Context(), cloneID, sessionID, artifactType)
	if err != nil {
		api.SendError(w, r, err)
		return
	}

	defer func() {
		if err := body.Close(); err != nil {
			log.Err()
		}
	}()

	if _, err := io.Copy(w, body); err != nil {
		api.SendError(w, r, errors.Wrapf(err, "failed to download artifact %q", artifactType))
		return
	}
}

func (s *Server) destroyClone(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	cloneID := values.Get("clone_id")

	if cloneID == "" {
		api.SendBadRequestError(w, r, "Clone ID must not be empty")
		return
	}

	if err := s.dle.DestroyClone(context.Background(), cloneID); err != nil {
		api.SendError(w, r, errors.Wrap(err, "failed to destroy clone"))
		return
	}

	log.Dbg(fmt.Sprintf("Clone ID=%s is being deleted", cloneID))
}

// healthCheck provides a health check handler.
func (s *Server) healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	healthResponse := models.Engine{
		Version: version.GetVersion(),
	}

	if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Err(err)

		return
	}
}

// readJSON reads JSON from request.
func readJSON(r *http.Request, v interface{}) error {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read a request body")
	}

	if err = json.Unmarshal(reqBody, v); err != nil {
		return errors.Wrapf(err, "failed to unmarshal json: %s", string(reqBody))
	}

	return nil
}
