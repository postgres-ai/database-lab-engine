package physical

import (
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
