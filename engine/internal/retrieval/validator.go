package retrieval

import (
	"errors"
	"fmt"
	"strings"

	"github.com/robfig/cron/v3"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/logical"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/physical"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/snapshot"
)

// ValidateConfig validates retrieval configuration.
func ValidateConfig(cfg *config.Config) (*config.Config, error) {
	retrievalCfg, err := formatJobsSpec(cfg)
	if err != nil {
		return nil, err
	}

	if err = validateRefreshTimetable(retrievalCfg); err != nil {
		return nil, err
	}

	if err = validateStructure(retrievalCfg); err != nil {
		return nil, err
	}

	return retrievalCfg, nil
}

// formatJobsSpec validates job list and enriches job specifications.
func formatJobsSpec(cfg *config.Config) (*config.Config, error) {
	jobSpecs := make(map[string]config.JobSpec, len(cfg.Jobs))
	undefinedJobs := []string{}

	for _, jobName := range cfg.Jobs {
		jobSpec, ok := cfg.JobsSpec[jobName]
		if !ok {
			undefinedJobs = append(undefinedJobs, jobName)
			continue
		}

		jobSpec.Name = jobName
		jobSpecs[jobName] = jobSpec
	}

	if len(undefinedJobs) > 0 {
		return nil, fmt.Errorf("config contains jobs without specification: %s", strings.Join(undefinedJobs, ", "))
	}

	jobsCfg := &config.Config{
		Refresh:  cfg.Refresh,
		Jobs:     cfg.Jobs,
		JobsSpec: jobSpecs,
	}

	return jobsCfg, nil
}

// validateStructure checks if the retrieval configuration is valid.
func validateStructure(r *config.Config) error {
	if hasLogicalJob(r.JobsSpec) && hasPhysicalJob(r.JobsSpec) {
		return errors.New("must not contain physical and logical jobs simultaneously")
	}

	return nil
}

func validateRefreshTimetable(r *config.Config) error {
	if r.Refresh == nil || r.Refresh.Timetable == "" {
		return nil
	}

	specParser := cron.NewParser(parseOption)

	_, err := specParser.Parse(r.Refresh.Timetable)
	if err != nil {
		return fmt.Errorf("invalid timetable: %w", err)
	}

	return nil
}

func hasLogicalJob(jobSpecs map[string]config.JobSpec) bool {
	if len(jobSpecs) == 0 {
		return false
	}

	if _, hasLogicalDump := jobSpecs[logical.DumpJobType]; hasLogicalDump {
		return true
	}

	if _, hasLogicalRestore := jobSpecs[logical.RestoreJobType]; hasLogicalRestore {
		return true
	}

	if _, hasLogicalSnapshot := jobSpecs[snapshot.LogicalSnapshotType]; hasLogicalSnapshot {
		return true
	}

	return false
}

func hasPhysicalJob(jobSpecs map[string]config.JobSpec) bool {
	if len(jobSpecs) == 0 {
		return false
	}

	if _, hasPhysicalRestore := jobSpecs[physical.RestoreJobType]; hasPhysicalRestore {
		return true
	}

	if _, hasPhysicalSnapshot := jobSpecs[snapshot.PhysicalSnapshotType]; hasPhysicalSnapshot {
		return true
	}

	return false
}
