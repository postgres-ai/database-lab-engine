/*
2022 © Postgres.ai
*/

// Package diagnostic collect containers logs in case of failures.
package diagnostic

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"github.com/robfig/cron/v3"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

// Config diagnostic configuration.
type Config struct {
	LogsRetentionDays int `yaml:"logsRetentionDays"`
}

// Cleaner logs cleanup job.
type Cleaner struct {
	cleanerCron *cron.Cron
}

const (
	cleanInterval       = "0 * * * *"
	timeFormat          = "20060102150405"
	containerOutputFile = "output.txt"

	defaultLogsRetentionDays = 7
)

// CollectDiagnostics collects container output and Postgres logs.
func CollectDiagnostics(ctx context.Context, client *client.Client, filterArgs filters.Args,
	postgresContainerName, dbDataDir string) error {
	diagnosticsDir, err := util.GetLogsPath(time.Now().Format(timeFormat))

	if err != nil {
		return fmt.Errorf("failed to get logs diagnostics directory %w", err)
	}

	if err := os.MkdirAll(diagnosticsDir, 0755); err != nil {
		return fmt.Errorf("failed to create diagnostics dir %s %w", diagnosticsDir, err)
	}

	err = collectContainersOutput(ctx, client, diagnosticsDir, filterArgs)

	if err != nil {
		return fmt.Errorf("failed to collect containers output %w", err)
	}

	err = collectPostgresLogs(ctx, client, diagnosticsDir, postgresContainerName, dbDataDir)

	if err != nil {
		return fmt.Errorf("failed to collect postgres logs %w", err)
	}

	return nil
}

// CollectContainerDiagnostics collect specific container diagnostics information.
func CollectContainerDiagnostics(ctx context.Context, client *client.Client, containerName, dbDataDir string) {
	diagnosticsDir, err := util.GetLogsPath(time.Now().Format(timeFormat))

	if err != nil {
		log.Warn("failed to get logs diagnostics directory %w", err)
		return
	}

	if err := os.MkdirAll(diagnosticsDir, 0755); err != nil {
		log.Warn("failed to create diagnostics dir %s %w", diagnosticsDir, err)
		return
	}

	err = collectContainerLogs(ctx, client, diagnosticsDir, containerName)
	if err != nil {
		log.Warn("failed to collect container logs ", containerName, err)
	}

	err = collectPostgresLogs(ctx, client, diagnosticsDir, containerName, dbDataDir)

	if err != nil {
		log.Warn("failed to collect Postgres logs ", containerName, err)
	}
}

func collectContainersOutput(ctx context.Context, client *client.Client, diagnosticDir string, filterArgs filters.Args) error {
	containerList, err := tools.ListContainersByLabel(ctx, client, filterArgs)

	if err != nil {
		return err
	}

	for _, containerName := range containerList {
		err = collectContainerLogs(ctx, client, diagnosticDir, containerName)
		if err != nil {
			log.Warn("failed to collect container logs ", containerName, err)
		}
	}

	return nil
}

func collectContainerLogs(ctx context.Context, client *client.Client, diagnosticDir string, containerName string) error {
	containerLogsDir := path.Join(diagnosticDir, containerName)

	if err := os.MkdirAll(containerLogsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory %w", err)
	}

	stdoutFile := path.Join(containerLogsDir, containerOutputFile)

	err := tools.CopyContainerLogs(ctx, client, containerName, stdoutFile)

	if err != nil {
		return fmt.Errorf("failed to get container logs %s %w", containerName, err)
	}

	return nil
}

// NewLogCleaner create new diagnostic cleaner.
func NewLogCleaner() *Cleaner {
	return &Cleaner{cleanerCron: cron.New()}
}

// ScheduleLogCleanupJob start background job for logs cleanup.
func (d *Cleaner) ScheduleLogCleanupJob(config Config) error {
	specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := specParser.Parse(cleanInterval)

	// validate configuration before scheduling cleaner

	if err != nil {
		return err
	}

	var logRetentionDays = config.LogsRetentionDays

	if logRetentionDays < 0 {
		return fmt.Errorf("invalid value for logsRetentionDays %d", logRetentionDays)
	}

	if logRetentionDays == 0 {
		logRetentionDays = defaultLogsRetentionDays
	}

	d.StopLogCleanupJob()

	d.cleanerCron.Schedule(schedule, cron.FuncJob(cleanLogsFunc(logRetentionDays)))
	d.cleanerCron.Start()

	return nil
}

// StopLogCleanupJob stop logs cleanup job.
func (d *Cleaner) StopLogCleanupJob() {
	if d.cleanerCron != nil {
		d.cleanerCron.Stop()
	}
}

func collectPostgresLogs(ctx context.Context, client *client.Client, diagnosticDir, dbContainerName, dbDataDir string) error {
	log.Dbg("Collecting postgres logs from container", dbContainerName, dbDataDir)
	containerLogsDir := path.Join(diagnosticDir, dbContainerName)

	if err := os.MkdirAll(containerLogsDir, 0755); err != nil {
		return fmt.Errorf("failed to create postgres logs directory %w", err)
	}

	// copy logs directory from container, result is a TAR stream
	// log directory is considered "dbDataDir/log"
	reader, _, err := client.CopyFromContainer(ctx, dbContainerName, fmt.Sprintf("%s/log/", dbDataDir))

	if err != nil {
		return fmt.Errorf("failed to copy postgres logs %w", err)
	}

	// process TAR stream and extract it as separated files
	tr := tar.NewReader(reader)

	for {
		header, err := tr.Next()

		// reached end of TAR stream
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Warn(err)
			continue
		}

		err = extractTar(containerLogsDir, tr, header)

		if err != nil {
			// log error and try to process next entry
			log.Err(err)
		}
	}

	return nil
}

func extractTar(dir string, reader *tar.Reader, header *tar.Header) error {
	target := filepath.Join(dir, header.Name)

	switch header.Typeflag {
	// create directory
	case tar.TypeDir:
		if _, err := os.Stat(target); err != nil {
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory from TAR stream %s %w", target, err)
			}
		}
	// extract file from TAR stream
	case tar.TypeReg:
		f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
		if err != nil {
			return fmt.Errorf("failed to create file from TAR stream %s %w", target, err)
		}

		defer func() {
			if err := f.Close(); err != nil {
				log.Err("failed to close TAR stream", err)
			}
		}()

		if _, err := io.Copy(f, reader); err != nil {
			return fmt.Errorf("failed to write TAR file %s %w", target, err)
		}
	}

	return nil
}

func cleanLogsFunc(logRetentionDays int) func() {
	return func() {
		logsDir, err := util.GetLogsRoot()

		log.Dbg("Cleaning old logs", logsDir)

		if err != nil {
			log.Err("failed to fetch logs dir", err)
			return
		}

		err = cleanupLogsDir(logsDir, logRetentionDays)

		if err != nil {
			log.Err("failed to fetch logs dir", err)
			return
		}
	}
}

func cleanupLogsDir(logsDir string, logRetentionDays int) error {
	// list log directories
	dirList, err := os.ReadDir(logsDir)

	if err != nil {
		log.Err("failed to list logs directories", err)
		return err
	}

	// directories "before" timeMark will be removed
	timeMark := time.Now().AddDate(0, 0, -1*logRetentionDays)

	for _, dir := range dirList {
		name := dir.Name()
		dirTime, err := time.Parse(timeFormat, name)

		if err != nil {
			log.Warn("failed to parse time", name, err)
			continue
		}

		if dirTime.After(timeMark) {
			continue
		}

		log.Dbg("Removing old logs directory", name)

		if err = os.RemoveAll(path.Join(logsDir, name)); err != nil {
			log.Err("directory removal failed", err)
		}
	}

	return nil
}
