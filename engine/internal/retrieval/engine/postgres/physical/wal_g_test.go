package physical

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWALGRecoveryConfig(t *testing.T) {
	walg := newWALG(nil, "dataDir", walgOptions{})

	recoveryConfig := walg.GetRecoveryConfig(11.7)
	expectedResponse11 := map[string]string{
		"restore_command":          "wal-g wal-fetch %f %p",
		"recovery_target_timeline": "latest",
	}
	assert.Equal(t, expectedResponse11, recoveryConfig)

	recoveryConfig = walg.GetRecoveryConfig(12.3)
	expectedResponse12 := map[string]string{
		"restore_command": "wal-g wal-fetch %f %p",
	}
	assert.Equal(t, expectedResponse12, recoveryConfig)
}

func TestWALGVersionParse(t *testing.T) {
	version, err := parseWalGVersion("wal-g version v2.0.0\t1eb88a5\t2022.05.20_10:45:57\tPostgreSQL")
	assert.NoError(t, err)
	assert.NotEmpty(t, version)
	assert.Equal(t, "v2.0.0", version)
}

func TestWALGGetRestoreCommand(t *testing.T) {
	testCases := []struct {
		name       string
		pgDataDir  string
		backupName string
		expected   string
	}{
		{name: "latest backup", pgDataDir: "/pgdata", backupName: "LATEST", expected: "wal-g backup-fetch /pgdata LATEST"},
		{name: "named backup", pgDataDir: "/var/lib/postgresql/data", backupName: "base_000000010000000000000005", expected: "wal-g backup-fetch /var/lib/postgresql/data base_000000010000000000000005"},
		{name: "empty backup name", pgDataDir: "/data", backupName: "", expected: "wal-g backup-fetch /data "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := newWALG(nil, tc.pgDataDir, walgOptions{BackupName: tc.backupName})
			assert.Equal(t, tc.expected, w.GetRestoreCommand())
		})
	}
}

func TestWALGGetRestoreCommand_CustomOptions(t *testing.T) {
	testCases := []struct {
		name          string
		customOptions []string
		expected      string
	}{
		{name: "no custom options", customOptions: nil, expected: "wal-g backup-fetch /pgdata LATEST"},
		{name: "empty custom options", customOptions: []string{}, expected: "wal-g backup-fetch /pgdata LATEST"},
		{name: "single option", customOptions: []string{"--reverse-unpack"}, expected: "wal-g backup-fetch /pgdata LATEST --reverse-unpack"},
		{name: "option with value", customOptions: []string{"--mask", "'base/*'"}, expected: "wal-g backup-fetch /pgdata LATEST --mask 'base/*'"},
		{name: "multiple options", customOptions: []string{"--reverse-unpack", "--skip-redundant-tars"}, expected: "wal-g backup-fetch /pgdata LATEST --reverse-unpack --skip-redundant-tars"},
		{name: "quotes are passed through unmodified", customOptions: []string{`--restore-only=my_db,"another db"`}, expected: `wal-g backup-fetch /pgdata LATEST --restore-only=my_db,"another db"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := newWALG(nil, "/pgdata", walgOptions{BackupName: "LATEST", CustomOptions: tc.customOptions})
			assert.Equal(t, tc.expected, w.GetRestoreCommand())
		})
	}
}

func TestWALGVersionParse_TableDriven(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{name: "v2.0.0", input: "wal-g version v2.0.0\t1eb88a5\t2022.05.20_10:45:57\tPostgreSQL", expected: "v2.0.0"},
		{name: "v1.1", input: "wal-g version v1.1\tabc1234\t2021.01.01_00:00:00\tPostgreSQL", expected: "v1.1"},
		{name: "v0.2.0", input: "wal-g version v0.2.0\tdef5678\t2020.06.15_12:00:00\tPostgreSQL", expected: "v0.2.0"},
		{name: "too few parts", input: "wal-g", hasError: true},
		{name: "only two parts", input: "wal-g version", hasError: true},
		{name: "empty string", input: "", hasError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			version, err := parseWalGVersion(tc.input)
			if tc.hasError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, version)
		})
	}
}

func TestWALGRecoveryConfig_VersionBoundary(t *testing.T) {
	w := newWALG(nil, "/pgdata", walgOptions{})

	testCases := []struct {
		name           string
		version        float64
		expectTimeline bool
	}{
		{name: "pg 9.6", version: 9.6, expectTimeline: true},
		{name: "pg 11.99", version: 11.99, expectTimeline: true},
		{name: "pg 12.0 exact", version: 12.0, expectTimeline: false},
		{name: "pg 14.0", version: 14.0, expectTimeline: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := w.GetRecoveryConfig(tc.version)
			assert.Equal(t, "wal-g wal-fetch %f %p", cfg["restore_command"])
			if tc.expectTimeline {
				assert.Equal(t, "latest", cfg["recovery_target_timeline"])
			} else {
				_, exists := cfg["recovery_target_timeline"]
				assert.False(t, exists)
			}
		})
	}
}

const walgDetailOutput = `INFO: 2026/10/21 11:17:27.000000 List backups from storages: [default]
name                                                     modified             wal_segment_backup_start start_time finish_time hostname data_dir
base_000000010000000000000004                            2026-10-20T10:00:00Z 000000010000000000000004 2026-10-20T10:00:00Z 2026-10-20T10:05:00Z db1 /var/lib/postgresql/base
base_000000010000000000000009_D_000000010000000000000004 2026-10-21T10:00:00Z 000000010000000000000009 2026-10-21T10:00:00Z 2026-10-21T10:01:00Z db1 /var/lib/postgresql/base
`

const walgListOutput = `name                          modified             wal_segment_backup_start
base_00000002000000000000000A 2026-10-21T10:00:00Z 00000002000000000000000A
base_000000010000000000000004 2026-10-20T10:00:00Z 000000010000000000000004
`

func TestWALGBackupNames(t *testing.T) {
	testCases := []struct {
		name            string
		output          string
		expectedNames   []string
		expectedLast    string
		expectedHighest string
	}{
		{name: "empty output", output: ""},
		{name: "no backups found", output: "INFO: 2026/10/21 11:17:27.000000 No backups found"},
		{name: "header only", output: "name modified wal_segment_backup_start"},
		{
			name: "detail output with log line", output: walgDetailOutput,
			expectedNames:   []string{"base_000000010000000000000004", "base_000000010000000000000009_D_000000010000000000000004"},
			expectedLast:    "base_000000010000000000000009_D_000000010000000000000004",
			expectedHighest: "base_000000010000000000000009_D_000000010000000000000004",
		},
		{
			name: "list output out of order", output: walgListOutput,
			expectedNames:   []string{"base_00000002000000000000000A", "base_000000010000000000000004"},
			expectedLast:    "base_000000010000000000000004",
			expectedHighest: "base_00000002000000000000000A",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedNames, backupNames(tc.output))
			assert.Equal(t, tc.expectedLast, lastListedBackup(tc.output))
			assert.Equal(t, tc.expectedHighest, highestBackupName(tc.output))
		})
	}
}

type execResult struct {
	output string
	err    error
}

func TestWALGInitLatestBackup(t *testing.T) {
	const (
		v2       = "wal-g version v2.0.0\t1eb88a5\t2022.05.20_10:45:57\tPostgreSQL"
		v0       = "wal-g version v0.2.19\t1eb88a5\t2021.01.01_10:00:00\tPostgreSQL"
		detail   = "wal-g backup-list --detail 2>&1"
		list     = "wal-g backup-list 2>&1"
		denied   = "ERROR: 2026/10/21 11:17:27.000000 AccessDenied: Access Denied status code: 403"
		noBackup = "INFO: 2026/10/21 11:17:27.000000 No backups found"
	)

	exitErr := errors.New("exit code: 1")

	testCases := []struct {
		name          string
		results       map[string]execResult
		expectedName  string
		expectedError string
	}{
		{
			name:         "detail output gives the last listed backup",
			results:      map[string]execResult{detail: {output: walgDetailOutput}},
			expectedName: "base_000000010000000000000009_D_000000010000000000000004",
		},
		{
			name:         "detail failure falls back to the plain list",
			results:      map[string]execResult{detail: {output: "unknown flag: --detail", err: exitErr}, list: {output: walgListOutput}},
			expectedName: "base_00000002000000000000000A",
		},
		{
			name:          "wal-g error output reaches the caller",
			results:       map[string]execResult{"wal-g --version": {output: v0}, list: {output: denied, err: exitErr}},
			expectedError: "failed to list wal-g backups: exit code: 1: " + denied,
		},
		{
			name:          "empty storage reports no backups",
			results:       map[string]execResult{detail: {output: noBackup}},
			expectedError: "no backups found in the configured storage",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results := map[string]execResult{"wal-g --version": {output: v2}}
			for cmd, res := range tc.results {
				results[cmd] = res
			}

			w := newWALG(nil, "dataDir", walgOptions{BackupName: "latest"})
			w.exec = func(_ context.Context, containerID, cmd string) (string, error) {
				res, ok := results[cmd]
				require.True(t, ok, "unexpected command %q in container %q", cmd, containerID)

				return res.output, res.err
			}

			err := w.Init(context.Background(), "containerID")
			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedName, w.parsedBackupName)
		})
	}
}
