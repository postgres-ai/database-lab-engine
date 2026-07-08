package srv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	imagetypes "github.com/docker/docker/api/types/image"
	yamlv2 "gopkg.in/yaml.v2"
	"gopkg.in/yaml.v3"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/physical"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/db"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/probe"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/projection"
	yamlUtils "gitlab.com/postgres-ai/database-lab/v3/pkg/util/yaml"
)

const (
	connectionCheckTimeout = 10 * time.Second
	configManagementDenied = "configuration management via UI/API disabled by admin"
)

func (s *Server) getProjectedAdminConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.projectedAdminConfig()
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := api.WriteJSON(w, http.StatusOK, cfg); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) getAdminConfigYaml(w http.ResponseWriter, r *http.Request) {
	cfg, err := adminConfigYaml()
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := api.WriteDataTyped(
		w,
		http.StatusOK,
		api.YamlContentType,
		cfg,
	); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) setProjectedAdminConfig(w http.ResponseWriter, r *http.Request) {
	if s.configModificationDisabled() {
		api.SendBadRequestError(w, r, configManagementDenied)
		return
	}

	var cfg interface{}
	if err := api.ReadJSON(r, &cfg); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	applied, err := s.applyProjectedAdminConfig(r.Context(), cfg)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	s.tm.SendEvent(context.Background(), telemetry.ConfigUpdatedEvent, telemetry.ConfigUpdated{})

	retrievalStatus := s.Retrieval.State.Status

	if err := s.Retrieval.RemovePendingMarker(); err != nil {
		api.SendError(w, r, err)
		return
	}

	if retrievalStatus == models.Pending {
		go func() {
			if err := s.Retrieval.FullRefresh(context.Background()); err != nil {
				log.Err(fmt.Errorf("failed to refresh data: %w", err))
			}
		}()
	}

	if err := api.WriteJSON(w, http.StatusOK, applied); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) testDBSource(w http.ResponseWriter, r *http.Request) {
	if s.configModificationDisabled() {
		api.SendBadRequestError(w, r, configManagementDenied)
		return
	}

	if s.Retrieval.State.Mode != models.Logical {
		api.SendBadRequestError(w, r, "the endpoint is only available in the Logical mode of the data retrieval")
		return
	}

	var connection models.ConnectionTest
	if err := api.ReadJSON(r, &connection); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := connectionPassword(&connection); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), connectionCheckTimeout)
	defer cancel()

	tc, err := db.CheckSource(ctx, &connection, s.Retrieval.ImageContent())
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := api.WriteJSON(w, http.StatusOK, tc); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) probeSource(w http.ResponseWriter, r *http.Request) {
	if s.Config.DisableConfigModification {
		api.SendBadRequestError(w, r, configManagementDenied)
		return
	}

	var req models.ProbeSourceRequest
	if err := api.ReadJSON(r, &req); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), connectionCheckTimeout)
	defer cancel()

	proposed, err := probe.Propose(ctx, req.URL, req.Password, s.imageRegistry)
	if err != nil {
		// matches testDBSource's "400 for input + connectivity" convention. the error message
		// is never the raw URL or password — Propose wraps with structured prefixes
		// (parse / connect to source / query ...).
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	s.tm.SendEvent(r.Context(), telemetry.ConfigProbedEvent, telemetry.ConfigProbed{
		Provider: string(proposed.DetectedProvider),
	})

	resp := models.ProposedConfig{
		Source: models.SourceConnection{
			Host:     proposed.Source.Host,
			Port:     proposed.Source.Port,
			Username: proposed.Source.Username,
			DBName:   proposed.Source.DBName,
		},
		DetectedProvider:       string(proposed.DetectedProvider),
		DockerImage:            proposed.DockerImage,
		DockerTag:              proposed.DockerTag,
		ResolvedImage:          proposed.ResolvedImage,
		PgMajorVersion:         proposed.PgMajorVersion,
		CollationVersion:       proposed.CollationVersion,
		Databases:              proposed.Databases,
		SharedBuffers:          proposed.SharedBuffers,
		MemoryProbed:           proposed.MemoryProbed,
		SharedPreloadLibraries: proposed.SharedPreloadLibraries,
		QueryTuning:            proposed.QueryTuning,
	}

	if err := api.WriteJSON(w, http.StatusOK, resp); err != nil {
		api.SendError(w, r, err)
		return
	}
}

// requestedRetrievalMode reads the synthetic `retrievalMode` field from the
// incoming projection JSON, falling back to the running retrieval state when
// the client omits it (older UI builds, direct API callers).
func requestedRetrievalMode(objMap map[string]interface{}, fallback models.RetrievalMode) models.RetrievalMode {
	raw, ok := objMap["retrievalMode"]
	if !ok {
		return fallback
	}

	asString, ok := raw.(string)
	if !ok || asString == "" {
		return fallback
	}

	return models.RetrievalMode(asString)
}

// guardModeFields rejects projections that mix logical-only and physical-only
// fields with the wrong mode. It protects hand-edited physical configs from
// being wiped by stale UI state, and vice-versa.
func guardModeFields(mode models.RetrievalMode, proj *models.ConfigProjection) error {
	logicalFields := []struct {
		name string
		set  bool
	}{
		{"connectionString", proj.ConnectionString != nil},
		{"host", proj.Host != nil},
		{"port", proj.Port != nil},
		{"username", proj.Username != nil},
		{"dbname", proj.DBName != nil},
		{"password", proj.Password != nil},
		{"databases", proj.DBList != nil},
		{"dumpParallelJobs", proj.DumpParallelJobs != nil},
		{"restoreParallelJobs", proj.RestoreParallelJobs != nil},
		{"restoreConfigs", proj.RestoreConfigs != nil},
		{"dumpCustomOptions", proj.DumpCustomOptions != nil},
		{"restoreCustomOptions", proj.RestoreCustomOptions != nil},
		{"ignoreDumpErrors", proj.IgnoreDumpErrors != nil},
		{"ignoreRestoreErrors", proj.IgnoreRestoreErrors != nil},
		{"rdsIamDbInstanceIdentifier", proj.RDSIAMDBInstance != nil},
	}

	physicalFields := []struct {
		name string
		set  bool
	}{
		{"physicalTool", proj.PhysicalTool != nil},
		{"physicalDockerImage", proj.PhysicalDockerImage != nil},
		{"physicalSyncEnabled", proj.PhysicalSyncEnabled != nil},
		{"physicalWalgBackupName", proj.PhysicalWalgBackupName != nil},
		{"physicalPgbackrestStanza", proj.PhysicalPgbackrestStanza != nil},
		{"physicalPgbackrestDelta", proj.PhysicalPgbackrestDelta != nil},
		{"physicalEnvs", proj.PhysicalEnvs != nil},
	}

	switch mode {
	case models.Logical:
		for _, f := range physicalFields {
			if f.set {
				return fmt.Errorf("logical-mode config update must not set physical-mode field %q", f.name)
			}
		}

	case models.Physical:
		for _, f := range logicalFields {
			if f.set {
				return fmt.Errorf("physical-mode config update must not set logical-mode field %q", f.name)
			}
		}
	}

	return nil
}

// validateSourceConnectionString rejects a source connection string that embeds
// a password (or is otherwise unparseable) before it is persisted to the config.
// The returned error never echoes the string, so a password placed in it cannot
// leak into logs or the API response. An empty string is a no-op: Expert-mode
// saves send it to clear a stale connection string written by the CLI, and the
// discrete connection.* fields then take effect.
func validateSourceConnectionString(connStr *string) error {
	if connStr == nil || *connStr == "" {
		return nil
	}

	_, err := probe.ParseConnectionString(*connStr)

	switch {
	case err == nil:
		return nil
	case errors.Is(err, probe.ErrPasswordInConnString):
		return probe.ErrPasswordInConnString
	case errors.Is(err, probe.ErrMultiHostConnString):
		return probe.ErrMultiHostConnString
	default:
		return errors.New("invalid source connection string")
	}
}

func connectionPassword(connection *models.ConnectionTest) error {
	if connection.Password != "" {
		return nil
	}

	proj := &models.ConfigProjection{}

	data, err := config.GetConfigBytes()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	node := &yaml.Node{}

	if err = yaml.Unmarshal(data, node); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err = projection.LoadYaml(proj, node, projection.LoadOptions{
		Groups: []string{"sensitive"},
	}); err != nil {
		return fmt.Errorf("failed to load config projection: %w", err)
	}

	if proj.Password != nil {
		connection.Password = *proj.Password
	}

	return nil
}

func adminConfigYaml() ([]byte, error) {
	data, err := config.GetConfigBytes()
	if err != nil {
		return nil, err
	}

	document := &yaml.Node{}

	err = yaml.Unmarshal(data, document)
	if err != nil {
		return nil, err
	}

	yamlUtils.DefaultConfigMask().Yaml(document)
	yamlUtils.TraverseNode(document)

	doc, err := yaml.Marshal(document)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func (s *Server) projectedAdminConfig() (interface{}, error) {
	data, err := config.GetConfigBytes()
	if err != nil {
		return nil, err
	}

	document := &yaml.Node{}

	err = yaml.Unmarshal(data, document)
	if err != nil {
		return nil, err
	}

	proj := &models.ConfigProjection{}

	err = projection.LoadYaml(proj, document, projection.LoadOptions{
		Groups: []string{"default"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load yaml config projection: %w", err)
	}

	obj := map[string]interface{}{}

	err = projection.StoreJSON(proj, obj, projection.StoreOptions{
		Groups: []string{"default"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to jsonify config projection: %w", err)
	}

	// retrievalMode is a synthetic field — it has no YAML counterpart, so the
	// projection layer never writes it. The UI needs it to choose the initial
	// tab and the right Expert sub-form, so populate it from the running
	// retrieval state.
	obj["retrievalMode"] = string(s.Retrieval.State.Mode)

	return obj, nil
}

func (s *Server) applyProjectedAdminConfig(ctx context.Context, obj interface{}) (interface{}, error) {
	objMap, ok := obj.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config must be an object: %T", obj)
	}

	mode := requestedRetrievalMode(objMap, s.Retrieval.State.Mode)

	completeLogicalPipeline := false

	switch mode {
	case models.Logical:
		if s.Retrieval.State.Mode == models.Physical {
			return nil, fmt.Errorf("cannot apply a logical config: the instance is configured for physical " +
				"retrieval; switch modes by editing the config manually")
		}

		completeLogicalPipeline = true

	case models.Physical:
		if _, err := s.Retrieval.GetStageSpec(physical.RestoreJobType); err == retrieval.ErrStageNotFound {
			return nil, fmt.Errorf("physicalRestore job is not enabled. Consider editing DLE config manually")
		}

	default:
		return nil, fmt.Errorf("config update requires retrievalMode to be logical or physical, got %q", mode)
	}

	proj := &models.ConfigProjection{}

	err := projection.LoadJSON(proj, objMap, projection.LoadOptions{
		Groups: []string{"default", "sensitive"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load json config projection: %w", err)
	}

	if err := guardModeFields(mode, proj); err != nil {
		return nil, err
	}

	if err := validateSourceConnectionString(proj.ConnectionString); err != nil {
		return nil, err
	}

	if proj.Password != nil && *proj.Password == "" {
		proj.Password = nil // Avoid storing empty password
	}

	if proj.DockerImage != nil && *proj.DockerImage == "" {
		proj.DockerImage = nil // avoid pulling or storing an empty image reference
	}

	data, err := config.GetConfigBytes()
	if err != nil {
		return nil, err
	}

	node := &yaml.Node{}

	err = yaml.Unmarshal(data, node)
	if err != nil {
		return nil, err
	}

	if completeLogicalPipeline {
		if err := ensureLogicalPipeline(node); err != nil {
			return nil, fmt.Errorf("failed to ensure logical retrieval pipeline: %w", err)
		}
	}

	err = projection.StoreYaml(proj, node, projection.StoreOptions{
		Groups: []string{"default", "sensitive"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to prepare yaml config projection: %w", err)
	}

	cfgData, err := yaml.Marshal(node)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(cfgData, data) {
		log.Msg("Config changed, validating...")

		err = s.validateConfig(ctx, proj, cfgData)
		if err != nil {
			return nil, err
		}

		log.Msg("Backing up config...")

		err = config.RotateConfig(cfgData)
		if err != nil {
			log.Errf("failed to backup config: %v", err)
			return nil, err
		}

		log.Msg("Config backed up successfully")
		log.Msg("Reloading configuration...")

		err = s.reloadFn(s)
		if err != nil {
			log.Msg("Failed to reload configuration", err)
			return nil, err
		}

		log.Msg("Configuration reloaded")
	} else {
		log.Msg("No changes detected in the config, skipping backup and reload")
	}

	result, err := s.projectedAdminConfig()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Server) validateConfig(
	ctx context.Context,
	proj *models.ConfigProjection,
	nodeBytes []byte,
) error {
	cfg := &config.Config{}

	// yamlv2 is used because v3 returns an error when config is deserialized
	err := yamlv2.Unmarshal(nodeBytes, cfg)
	if err != nil {
		return err
	}

	// Validating unmarshalled config is better because it represents actual usage
	err = provision.IsValidConfig(cfg.Provision)
	if err != nil {
		return err
	}

	_, err = retrieval.ValidateConfig(&cfg.Retrieval)
	if err != nil {
		return err
	}

	if err := validateCustomOptions(proj.DumpCustomOptions); err != nil {
		return fmt.Errorf("invalid custom dump options: %w", err)
	}

	if err := validateCustomOptions(proj.RestoreCustomOptions); err != nil {
		return fmt.Errorf("invalid custom restore options: %w", err)
	}

	if proj.DockerImage != nil {
		stream, err := s.docker.ImagePull(ctx, *proj.DockerImage, imagetypes.PullOptions{})
		if err != nil {
			return err
		}

		err = stream.Close()
		if err != nil {
			log.Err(err)
		}
	}

	return nil
}

var (
	isValidCustomOption = regexp.MustCompile("^[A-Za-z0-9-_=\"]+$").MatchString
	errInvalidOption    = fmt.Errorf("due to security reasons, current implementation of custom options supports only " +
		"letters, numbers, hyphen, underscore, equal sign, and double quotes")
	errInvalidOptionType = fmt.Errorf("invalid type of custom option")
)

func validateCustomOptions(customOptions []interface{}) error {
	for _, opt := range customOptions {
		castedValue, ok := opt.(string)
		if !ok {
			return fmt.Errorf("%w: %q", errInvalidOptionType, opt)
		}

		if !isValidCustomOption(castedValue) {
			return fmt.Errorf("invalid option %q: %w", castedValue, errInvalidOption)
		}
	}

	return nil
}
