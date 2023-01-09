package srv

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/docker/docker/api/types"
	yamlv2 "gopkg.in/yaml.v2"
	"gopkg.in/yaml.v3"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/logical"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/db"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
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

	applied, err := s.applyProjectedAdminConfig(r.Context(), cfg)
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

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
	if s.Retrieval.State.Mode != models.Logical {
		return nil, fmt.Errorf("config is only available in logical mode")
	}

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

func (s *Server) applyProjectedAdminConfig(ctx context.Context, obj interface{}) (interface{}, error) {
	if s.Retrieval.State.Mode != models.Logical {
		return nil, fmt.Errorf("config is only available in logical mode")
	}

	if _, err := s.Retrieval.GetStageSpec(logical.DumpJobType); err == retrieval.ErrStageNotFound {
		return nil, fmt.Errorf("logicalDump job is not enabled. Consider editing DLE config manually")
	}

	objMap, ok := obj.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config must be an object: %T", obj)
	}

	proj := &models.ConfigProjection{}

	err := projection.LoadJSON(proj, objMap, projection.LoadOptions{
		Groups: []string{"default", "sensitive"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load json config projection: %w", err)
	}

	if proj.Password != nil && *proj.Password == "" {
		proj.Password = nil // Avoid storing empty password
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
			log.Errf("Failed to backup config: %v", err)
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
		stream, err := s.docker.ImagePull(ctx, *proj.DockerImage, types.ImagePullOptions{})
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
