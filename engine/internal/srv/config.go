package srv

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	imagetypes "github.com/docker/docker/api/types/image"
	yamlv2 "gopkg.in/yaml.v2"
	"gopkg.in/yaml.v3"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/logical"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/db"
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
	if s.Config.DisableConfigModification {
		api.SendBadRequestError(w, r, configManagementDenied)
		return
	}

	var cfg interface{}
	if err := api.ReadJSON(r, &cfg); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	response, err := s.applyProjectedAdminConfig(r.Context(), cfg)
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

	if err := api.WriteJSON(w, http.StatusOK, response); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) testDBSource(w http.ResponseWriter, r *http.Request) {
	if s.Config.DisableConfigModification {
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

	return obj, nil
}

func (s *Server) applyProjectedAdminConfig(ctx context.Context, obj interface{}) (*models.ConfigUpdateResponse, error) {
	objMap, ok := obj.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config must be an object: %T", obj)
	}

	// Check for retrieval-specific settings in logical mode
	if hasRetrievalSettings(objMap) {
		if s.Retrieval.State.Mode != models.Logical {
			return nil, fmt.Errorf("retrieval settings are only available in logical mode")
		}

		if _, err := s.Retrieval.GetStageSpec(logical.DumpJobType); err == retrieval.ErrStageNotFound {
			return nil, fmt.Errorf("logicalDump job is not enabled. Consider editing DLE config manually")
		}
	}

	// Load current config for comparison
	data, err := config.GetConfigBytes()
	if err != nil {
		return nil, err
	}

	currentNode := &yaml.Node{}
	if err = yaml.Unmarshal(data, currentNode); err != nil {
		return nil, err
	}

	currentProj := &models.ConfigProjection{}
	if err = projection.LoadYaml(currentProj, currentNode, projection.LoadOptions{
		Groups: []string{"default", "sensitive"},
	}); err != nil {
		return nil, fmt.Errorf("failed to load current config projection: %w", err)
	}

	// Load new config from request
	newProj := &models.ConfigProjection{}
	if err = projection.LoadJSON(newProj, objMap, projection.LoadOptions{
		Groups: []string{"default", "sensitive"},
	}); err != nil {
		return nil, fmt.Errorf("failed to load json config projection: %w", err)
	}

	if newProj.Password != nil && *newProj.Password == "" {
		newProj.Password = nil // avoid storing empty password
	}

	// Detect changes that require restart
	changedSettings, restartSettings := detectConfigChanges(currentProj, newProj)
	warnings := generateRestartWarnings(restartSettings)

	node := &yaml.Node{}
	if err = yaml.Unmarshal(data, node); err != nil {
		return nil, err
	}

	if err = projection.StoreYaml(newProj, node, projection.StoreOptions{
		Groups: []string{"default", "sensitive"},
	}); err != nil {
		return nil, fmt.Errorf("failed to prepare yaml config projection: %w", err)
	}

	cfgData, err := yaml.Marshal(node)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(cfgData, data) {
		log.Msg("Config changed, validating...")

		if err = s.validateConfig(ctx, newProj, cfgData); err != nil {
			return nil, err
		}

		log.Msg("Backing up config...")

		if err = config.RotateConfig(cfgData); err != nil {
			log.Errf("failed to backup config: %v", err)
			return nil, err
		}

		log.Msg("Config backed up successfully")
		log.Msg("Reloading configuration...")

		if err = s.reloadFn(s); err != nil {
			log.Msg("Failed to reload configuration", err)
			return nil, err
		}

		log.Msg("Configuration reloaded")

		if len(restartSettings) > 0 {
			log.Msg("Some settings require restart to take full effect:", restartSettings)
		}
	} else {
		log.Msg("No changes detected in the config, skipping backup and reload")
	}

	result, err := s.projectedAdminConfig()
	if err != nil {
		return nil, err
	}

	return &models.ConfigUpdateResponse{
		Config:          result,
		Warnings:        warnings,
		RequiresRestart: len(restartSettings) > 0,
		ChangedSettings: changedSettings,
		RestartSettings: restartSettings,
	}, nil
}

// hasRetrievalSettings checks if the config update contains retrieval-specific settings.
func hasRetrievalSettings(objMap map[string]interface{}) bool {
	retrievalKeys := []string{
		"host", "port", "dbname", "username", "password",
		"databases", "dumpParallelJobs", "restoreParallelJobs",
		"dumpCustomOptions", "restoreCustomOptions",
		"ignoreDumpErrors", "ignoreRestoreErrors", "timetable",
	}

	for _, key := range retrievalKeys {
		if _, ok := objMap[key]; ok {
			return true
		}
	}

	return false
}

// detectConfigChanges compares old and new configs to find changed settings.
func detectConfigChanges(oldProj, newProj *models.ConfigProjection) ([]string, []string) {
	var changedSettings, restartSettings []string

	oldVal := reflect.ValueOf(oldProj).Elem()
	newVal := reflect.ValueOf(newProj).Elem()
	projType := oldVal.Type()

	for i := 0; i < projType.NumField(); i++ {
		field := projType.Field(i)
		oldField := oldVal.Field(i)
		newField := newVal.Field(i)

		// Get the proj tag to determine the setting path
		projTag := field.Tag.Get("proj")
		if projTag == "" {
			continue
		}

		// Extract setting path (remove options like ,createKey)
		settingPath := strings.Split(projTag, ",")[0]

		// Check if field changed
		if !fieldEqual(oldField, newField) {
			changedSettings = append(changedSettings, settingPath)

			// Check if it requires restart
			restartTag := field.Tag.Get("restart")
			if restartTag == "true" {
				restartSettings = append(restartSettings, settingPath)
			}
		}
	}

	return changedSettings, restartSettings
}

// fieldEqual compares two reflect.Value instances for equality.
func fieldEqual(a, b reflect.Value) bool {
	if a.Kind() == reflect.Ptr && b.Kind() == reflect.Ptr {
		if a.IsNil() && b.IsNil() {
			return true
		}

		if a.IsNil() || b.IsNil() {
			return false
		}

		return reflect.DeepEqual(a.Elem().Interface(), b.Elem().Interface())
	}

	return reflect.DeepEqual(a.Interface(), b.Interface())
}

// generateRestartWarnings creates warning messages for settings that require restart.
func generateRestartWarnings(restartSettings []string) []models.ConfigWarning {
	warnings := make([]models.ConfigWarning, 0, len(restartSettings))

	for _, setting := range restartSettings {
		message, ok := models.RestartRequiredSettings[setting]
		if !ok {
			message = fmt.Sprintf("Changing %s requires a restart to take effect", setting)
		}

		warnings = append(warnings, models.ConfigWarning{
			Setting: setting,
			Message: message,
			Type:    "restart",
		})
	}

	return warnings
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
	if err = provision.IsValidConfig(cfg.Provision); err != nil {
		return fmt.Errorf("invalid provision config: %w", err)
	}

	// Validate retrieval config only if it's being used
	if s.Retrieval.State.Mode == models.Logical {
		if _, err = retrieval.ValidateConfig(&cfg.Retrieval); err != nil {
			return fmt.Errorf("invalid retrieval config: %w", err)
		}
	}

	if err = validateCustomOptions(proj.DumpCustomOptions); err != nil {
		return fmt.Errorf("invalid custom dump options: %w", err)
	}

	if err = validateCustomOptions(proj.RestoreCustomOptions); err != nil {
		return fmt.Errorf("invalid custom restore options: %w", err)
	}

	// Validate cloning settings
	if err = validateCloningSettings(proj); err != nil {
		return fmt.Errorf("invalid cloning config: %w", err)
	}

	// Validate port pool settings
	if err = validatePortPoolSettings(proj); err != nil {
		return fmt.Errorf("invalid port pool config: %w", err)
	}

	// Validate diagnostic settings
	if err = validateDiagnosticSettings(proj); err != nil {
		return fmt.Errorf("invalid diagnostic config: %w", err)
	}

	// Validate embedded UI settings
	if err = validateEmbeddedUISettings(proj); err != nil {
		return fmt.Errorf("invalid embedded UI config: %w", err)
	}

	// Validate webhook settings
	if err = validateWebhookSettings(proj); err != nil {
		return fmt.Errorf("invalid webhook config: %w", err)
	}

	if proj.DockerImage != nil {
		stream, err := s.docker.ImagePull(ctx, *proj.DockerImage, imagetypes.PullOptions{})
		if err != nil {
			return fmt.Errorf("failed to pull docker image: %w", err)
		}

		if err = stream.Close(); err != nil {
			log.Err(err)
		}
	}

	return nil
}

// validateCloningSettings validates cloning-related configuration.
func validateCloningSettings(proj *models.ConfigProjection) error {
	if proj.AccessHost != nil && *proj.AccessHost == "" {
		return fmt.Errorf("accessHost cannot be empty when specified")
	}

	return nil
}

// validatePortPoolSettings validates port pool configuration.
func validatePortPoolSettings(proj *models.ConfigProjection) error {
	if proj.PortPoolFrom != nil && proj.PortPoolTo != nil {
		if *proj.PortPoolFrom == 0 {
			return fmt.Errorf("portPool.from must be greater than 0")
		}

		if *proj.PortPoolTo == 0 {
			return fmt.Errorf("portPool.to must be greater than 0")
		}

		if *proj.PortPoolTo < *proj.PortPoolFrom {
			return fmt.Errorf("portPool.to must be greater than or equal to portPool.from")
		}
	}

	return nil
}

// validateDiagnosticSettings validates diagnostic configuration.
func validateDiagnosticSettings(proj *models.ConfigProjection) error {
	if proj.LogsRetentionDays != nil && *proj.LogsRetentionDays < 0 {
		return fmt.Errorf("logsRetentionDays must be a non-negative number")
	}

	return nil
}

// validateEmbeddedUISettings validates embedded UI configuration.
func validateEmbeddedUISettings(proj *models.ConfigProjection) error {
	if proj.EmbeddedUIPort != nil {
		if *proj.EmbeddedUIPort < 1 || *proj.EmbeddedUIPort > 65535 {
			return fmt.Errorf("embeddedUI.port must be between 1 and 65535")
		}
	}

	return nil
}

// validateWebhookSettings validates webhook configuration.
func validateWebhookSettings(proj *models.ConfigProjection) error {
	for i, hook := range proj.WebhooksHooks {
		if hook.URL == "" {
			return fmt.Errorf("webhook[%d].url cannot be empty", i)
		}

		if len(hook.Trigger) == 0 {
			return fmt.Errorf("webhook[%d].trigger cannot be empty", i)
		}

		validTriggers := map[string]bool{
			"clone.created": true,
			"clone.reset":   true,
			"clone.deleted": true,
		}

		for _, trigger := range hook.Trigger {
			if !validTriggers[trigger] {
				return fmt.Errorf("webhook[%d] has invalid trigger: %s", i, trigger)
			}
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
