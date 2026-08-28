/*
2026 © Postgres.ai
*/

package cloning

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/resources"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestLocaleProvider(t *testing.T) {
	tests := []struct {
		name     string
		row      map[string]any
		expected string
		wantErr  string
	}{
		{name: "pre-15 source has no provider column", row: map[string]any{}, expected: ""},
		{name: "libc", row: map[string]any{"datlocprovider": "c"}, expected: localeProviderLibc},
		{name: "icu", row: map[string]any{"datlocprovider": "i"}, expected: localeProviderICU},
		{name: "builtin is refused", row: map[string]any{"datlocprovider": "b"}, wantErr: "builtin locale provider"},
		{name: "unknown code is refused", row: map[string]any{"datlocprovider": "z"}, wantErr: "unknown locale provider"},
		{name: "null column behaves like absent", row: map[string]any{"datlocprovider": nil}, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := localeProvider(tt.row)

			if tt.wantErr == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, provider)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestICULocale(t *testing.T) {
	tests := []struct {
		name     string
		row      map[string]any
		expected string
	}{
		{name: "15 and 16 expose daticulocale", row: map[string]any{"daticulocale": "en-US"}, expected: "en-US"},
		{name: "17 renamed it to datlocale", row: map[string]any{"datlocale": "en-US"}, expected: "en-US"},
		{name: "empty daticulocale falls through to datlocale", row: map[string]any{"daticulocale": "", "datlocale": "fr-FR"}, expected: "fr-FR"},
		{name: "neither column present", row: map[string]any{}, expected: ""},
		{name: "null values", row: map[string]any{"daticulocale": nil, "datlocale": nil}, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, icuLocale(tt.row))
		})
	}
}

func TestUpgradeStatus(t *testing.T) {
	t.Run("success is OK and names the new version", func(t *testing.T) {
		status := upgradeStatus(provision.UpgradeResult{Outcome: provision.UpgradeSucceeded, Version: "17"})

		assert.Equal(t, models.StatusOK, status.Code)
		assert.Contains(t, status.Message, "PostgreSQL 17")
	})

	t.Run("rollback warns and points at the log", func(t *testing.T) {
		status := upgradeStatus(provision.UpgradeResult{
			Outcome: provision.UpgradeRolledBack, Version: "16",
			Cause: "pg_upgrade failed at stage \"check\"", LogPath: "/pool/c1/upgrade_logs/pg_upgrade_clone.log",
		})

		assert.Equal(t, models.StatusWarning, status.Code)
		assert.Contains(t, status.Message, "still running on PostgreSQL 16")
		assert.Contains(t, status.Message, "/pool/c1/upgrade_logs/pg_upgrade_clone.log")
	})

	t.Run("reprovision warns about lost data", func(t *testing.T) {
		status := upgradeStatus(provision.UpgradeResult{
			Outcome: provision.UpgradeReprovisioned, Version: "16", Cause: "exit code 20",
		})

		assert.Equal(t, models.StatusWarning, status.Code)
		assert.Contains(t, status.Message, "restored from its snapshot")
		assert.Contains(t, status.Message, "lost")
	})

	t.Run("a clone with nothing to recover goes back to OK", func(t *testing.T) {
		status := upgradeStatus(provision.UpgradeResult{Outcome: provision.UpgradeUnchanged, Version: "16"})

		assert.Equal(t, models.StatusOK, status.Code)
		assert.Equal(t, models.CloneMessageOK, status.Message)
	})

	t.Run("an empty outcome means recovery failed, which is the only FATAL", func(t *testing.T) {
		status := upgradeStatus(provision.UpgradeResult{Cause: "could not be restored"})

		assert.Equal(t, models.StatusFatal, status.Code)
		assert.Contains(t, status.Message, "could not be restored")
	})

	t.Run("a reprovisioned clone carries the log tail because its log was destroyed", func(t *testing.T) {
		status := upgradeStatus(provision.UpgradeResult{
			Outcome: provision.UpgradeReprovisioned, Version: "16", Cause: "exit code 20",
			LogTail: "pg_upgrade: error: could not link",
		})

		assert.Equal(t, models.StatusWarning, status.Code)
		assert.Contains(t, status.Message, "Log tail: pg_upgrade: error: could not link")
	})

	t.Run("a rollback of a clone with no recorded version names no version", func(t *testing.T) {
		status := upgradeStatus(provision.UpgradeResult{
			Outcome: provision.UpgradeRolledBack, Cause: "failed to connect to the clone",
		})

		assert.Equal(t, models.StatusWarning, status.Code)
		assert.Contains(t, status.Message, "still running on its original version")
		assert.NotContains(t, status.Message, "PostgreSQL .")
	})

	t.Run("a rollback with neither log path nor tail stays readable", func(t *testing.T) {
		status := upgradeStatus(provision.UpgradeResult{
			Outcome: provision.UpgradeRolledBack, Version: "16", Cause: "cannot prepare docker image",
		})

		assert.Equal(t, models.StatusWarning, status.Code)
		assert.Contains(t, status.Message, "cannot prepare docker image")
		assert.NotContains(t, status.Message, "See ")
	})
}

func TestUpgradeFailureStatus(t *testing.T) {
	t.Run("a failure the next start can finish must not be FATAL", func(t *testing.T) {
		status := upgradeFailureStatus(provision.UpgradeResult{Recoverable: true},
			errors.New("failed to start the upgraded clone"))

		assert.Equal(t, models.StatusWarning, status.Code,
			"FATAL costs the clone the dataset its upgrade already converted")
		assert.Contains(t, status.Message, models.CloneMessageUpgradeUnsettled)
		assert.Contains(t, status.Message, "failed to start the upgraded clone")
		assert.Contains(t, status.Message, "when the engine restarts")
	})

	t.Run("a failure with no upgrade state left is FATAL", func(t *testing.T) {
		status := upgradeFailureStatus(provision.UpgradeResult{}, errors.New("failed to find filesystem manager"))

		assert.Equal(t, models.StatusFatal, status.Code)
		assert.Equal(t, "failed to find filesystem manager", status.Message)
	})

	t.Run("the wrapped cause is unwrapped for the user", func(t *testing.T) {
		status := upgradeFailureStatus(provision.UpgradeResult{},
			errors.Wrap(errors.New("no such container"), "failed to remove the clone container"))

		assert.Equal(t, "no such container", status.Message)
	})
}

func TestLogReference(t *testing.T) {
	tests := []struct {
		name     string
		result   provision.UpgradeResult
		expected string
	}{
		{name: "log path wins when the dataset survived", result: provision.UpgradeResult{LogPath: "/pool/c1/upgrade/logs/x.log", LogTail: "tail"}, expected: "See /pool/c1/upgrade/logs/x.log"},
		{name: "tail is used when the dataset is gone", result: provision.UpgradeResult{LogTail: "tail"}, expected: "Log tail: tail"},
		{name: "nothing to reference", result: provision.UpgradeResult{}, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, logReference(tt.result))
		})
	}
}

func TestApplyUpgradeVersion(t *testing.T) {
	tests := []struct {
		name          string
		clone         *models.Clone
		result        provision.UpgradeResult
		expectedImage string
		expectedVer   string
	}{
		{
			name:  "success records the target image and version",
			clone: &models.Clone{ID: "c1"},
			result: provision.UpgradeResult{
				Outcome: provision.UpgradeSucceeded, Image: "postgresai/extended-postgres:17-0.8.0", Version: "17",
			},
			expectedImage: "postgresai/extended-postgres:17-0.8.0", expectedVer: "17",
		},
		{
			name:          "rollback leaves the clone untouched",
			clone:         &models.Clone{ID: "c1", DockerImage: "postgresai/extended-postgres:16-0.8.0", DBVersion: "16"},
			result:        provision.UpgradeResult{Outcome: provision.UpgradeRolledBack, Version: "16"},
			expectedImage: "postgresai/extended-postgres:16-0.8.0", expectedVer: "16",
		},
		{
			name:          "rollback on a never-upgraded clone keeps the override empty",
			clone:         &models.Clone{ID: "c1"},
			result:        provision.UpgradeResult{Outcome: provision.UpgradeRolledBack, Version: "16"},
			expectedImage: "", expectedVer: "",
		},
		{
			name:          "a pending reprovision request changes nothing yet",
			clone:         &models.Clone{ID: "c1", DockerImage: "postgresai/extended-postgres:17-0.8.0", DBVersion: "17"},
			result:        provision.UpgradeResult{Outcome: provision.UpgradeRequiresReprovision, Version: "16"},
			expectedImage: "postgresai/extended-postgres:17-0.8.0", expectedVer: "17",
		},
		{
			name:          "reprovision clears the override because the clone is back on the default",
			clone:         &models.Clone{ID: "c1", DockerImage: "postgresai/extended-postgres:17-0.8.0", DBVersion: "17"},
			result:        provision.UpgradeResult{Outcome: provision.UpgradeReprovisioned, Version: "16"},
			expectedImage: "", expectedVer: "",
		},
		{
			name:          "an unchanged clone takes the version its data directory reports",
			clone:         &models.Clone{ID: "c1", DockerImage: "postgresai/extended-postgres:17-0.8.0", DBVersion: "16"},
			result:        provision.UpgradeResult{Outcome: provision.UpgradeUnchanged, Version: "17"},
			expectedImage: "postgresai/extended-postgres:17-0.8.0", expectedVer: "17",
		},
		{
			name:          "an unchanged clone with an unreadable data directory keeps what it had",
			clone:         &models.Clone{ID: "c1", DockerImage: "postgresai/extended-postgres:16-0.8.0", DBVersion: "16"},
			result:        provision.UpgradeResult{Outcome: provision.UpgradeUnchanged},
			expectedImage: "postgresai/extended-postgres:16-0.8.0", expectedVer: "16",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Base{clones: make(map[string]*CloneWrapper)}
			wrapper := &CloneWrapper{Clone: tt.clone}

			c.applyUpgradeVersion(wrapper, tt.result)

			assert.Equal(t, tt.expectedImage, wrapper.Clone.DockerImage)
			assert.Equal(t, tt.expectedVer, wrapper.Clone.DBVersion)
		})
	}
}

// TestHasUnsettledUpgrade covers the wiring the snapshot guards depend on. The on-disk half is
// tested at the provision layer (TestHasPendingUpgrade); what is tested here is that this method
// answers true for a clone the engine is actively upgrading, and that it never reports true for a
// clone it cannot inspect - a snapshot guard that silently answers false is the failure mode.
func TestHasUnsettledUpgrade(t *testing.T) {
	newBase := func(w *CloneWrapper) *Base {
		base := &Base{clones: make(map[string]*CloneWrapper)}
		if w != nil {
			base.clones["c1"] = w
		}

		return base
	}

	upgrading := &CloneWrapper{
		Clone:   &models.Clone{ID: "c1", Status: models.Status{Code: models.StatusUpgrading}},
		Session: &resources.Session{},
	}

	t.Run("an unknown clone has nothing in flight", func(t *testing.T) {
		assert.False(t, newBase(nil).HasUnsettledUpgrade("c1"))
	})

	t.Run("a clone being upgraded right now", func(t *testing.T) {
		assert.True(t, newBase(upgrading).HasUnsettledUpgrade("c1"),
			"the status alone settles it while the upgrade goroutine is live")
	})

	t.Run("a clone with no session cannot be inspected", func(t *testing.T) {
		w := &CloneWrapper{Clone: &models.Clone{ID: "c1", Status: models.Status{Code: models.StatusOK}}}
		assert.False(t, newBase(w).HasUnsettledUpgrade("c1"))
	})
}
