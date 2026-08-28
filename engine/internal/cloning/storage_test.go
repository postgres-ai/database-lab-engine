package cloning

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/runners"
	"gitlab.com/postgres-ai/database-lab/v3/internal/telemetry"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

const (
	testingCloneState = `
{
  "c5bfsk0hmvjd7kau71jg": {
    "clone": {
      "id": "c5bfsk0hmvjd7kau71jg",
      "snapshot": {
        "id": "east5@snapshot_20211001112229",
        "createdAt": "2021-10-01T11:23:11+00:00",
        "dataStateAt": "2021-10-01T11:22:29+00:00"
      },
      "protected": false,
      "deleteAt": "",
      "createdAt": "2021-10-01T12:25:52+00:00",
      "status": {
        "code": "OK",
        "message": "Clone is ready to accept Postgres connections."
      },
      "db": {
        "connStr": "host=localhost port=6003 user=john dbname=postgres",
        "host": "localhost",
        "port": "6003",
        "username": "john",
        "password": "",
        "db_name": ""
      },
      "metadata": {
        "cloneDiffSize": 0,
        "cloneDiffSizeHR": "",
        "cloningTime": 0.749808646,
        "maxIdleMinutes": 10
      }
    },
    "session": {
      "id": "1",
      "pool": "east5",
      "port": 6003,
      "user": "postgres",
      "socket_host": "/var/lib/dblab/east5/sockets/dblab_clone_6003",
      "ephemeral_user": {
        "name": "john",
        "password": "test",
        "restricted": false,
        "available_db": ""
      },
      "extra_config": {}
    },
    "time_created_at": "2021-10-01T12:25:52.285539333Z",
    "time_started_at": "2021-10-01T12:25:53.03534795Z"
  }
}`
)

func prepareStateFile(data string) (string, error) {
	f, err := os.CreateTemp("", "dblab-clone-state-test-*.json")
	if err != nil {
		return "", err
	}

	if _, err := f.WriteString(data); err != nil {
		return "", err
	}

	return f.Name(), f.Close()
}

func newProvisioner() (*provision.Provisioner, error) {
	return provision.New(context.Background(), &provision.Config{
		PortPool: provision.PortPool{
			From: 1,
			To:   5,
		},
	}, nil, nil, pool.NewPoolManager(&pool.Config{}, runners.NewLocalRunner(false)), "instID", "nwID", "")
}

func TestLoadingSessionState(t *testing.T) {
	t.Run("it shouldn't panic if a state file is absent", func(t *testing.T) {
		s := &Base{}
		err := s.loadSessionState("/tmp/absent_session_file.json")
		assert.NoError(t, err)
	})

	t.Run("it loads sessions.json", func(t *testing.T) {
		filepath, err := prepareStateFile(testingCloneState)
		assert.NoError(t, err)

		defer func() { _ = os.Remove(filepath) }()

		s := &Base{}
		err = s.loadSessionState(filepath)
		assert.NoError(t, err)

		t.Run("it should restore valid clone's data", func(t *testing.T) {
			assert.Equal(t, 1, s.lenClones())
			w, ok := s.findWrapper("c5bfsk0hmvjd7kau71jg")
			require.True(t, ok)
			assert.Equal(t, "east5@snapshot_20211001112229", w.Clone.Snapshot.ID)
			assert.Equal(t, "east5", w.Session.Pool)
			assert.Equal(t, uint(6003), w.Session.Port)
		})
	})
}

func TestSavingSessionState(t *testing.T) {
	t.Run("it should save even if a clone list is empty", func(t *testing.T) {
		f, err := os.CreateTemp("", "dblab-clone-state-test-*.json")
		assert.NoError(t, err)
		defer func() { _ = os.Remove(f.Name()) }()

		prov, err := newProvisioner()
		assert.NoError(t, err)

		s := NewBase(nil, nil, prov, &telemetry.Agent{}, nil, nil)
		err = s.saveClonesState(f.Name())
		assert.NoError(t, err)

		data, err := os.ReadFile(f.Name())
		assert.NoError(t, err)

		assert.Equal(t, "{}", string(data))
	})
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsPath := tmpDir + "/sessions.json"

	prov, err := newProvisioner()
	require.NoError(t, err)

	base := NewBase(nil, nil, prov, &telemetry.Agent{}, nil, nil)
	base.setWrapper("clone1", &CloneWrapper{
		Clone: &models.Clone{
			ID:       "clone1",
			Status:   models.Status{Code: models.StatusOK, Message: models.CloneMessageOK},
			Snapshot: &models.Snapshot{ID: "snap1"},
			DB:       models.Database{Username: "testuser", DBName: "testdb", Port: "6000", Host: "localhost"},
		},
	})
	base.setWrapper("clone2", &CloneWrapper{
		Clone: &models.Clone{
			ID:     "clone2",
			Status: models.Status{Code: models.StatusCreating, Message: models.CloneMessageCreating},
		},
	})

	err = base.saveClonesState(sessionsPath)
	require.NoError(t, err)

	restored := &Base{}
	err = restored.loadSessionState(sessionsPath)
	require.NoError(t, err)

	assert.Equal(t, 2, restored.lenClones())

	w1, ok1 := restored.findWrapper("clone1")
	require.True(t, ok1)

	w2, ok2 := restored.findWrapper("clone2")
	require.True(t, ok2)

	assert.Equal(t, "clone1", w1.Clone.ID)
	assert.Equal(t, "snap1", w1.Clone.Snapshot.ID)
	assert.Equal(t, "testuser", w1.Clone.DB.Username)
	assert.Equal(t, models.StatusCreating, w2.Clone.Status.Code)
}

func TestLoadSessionStateInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsPath := tmpDir + "/sessions.json"

	err := os.WriteFile(sessionsPath, []byte("not valid json"), 0600)
	require.NoError(t, err)

	base := &Base{}
	err = base.loadSessionState(sessionsPath)
	assert.Error(t, err)
}

func TestSaveClonesStateFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsPath := tmpDir + "/sessions.json"

	prov, err := newProvisioner()
	require.NoError(t, err)

	base := NewBase(nil, nil, prov, &telemetry.Agent{}, nil, nil)
	err = base.saveClonesState(sessionsPath)
	require.NoError(t, err)

	info, err := os.Stat(sessionsPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestFilter(t *testing.T) {
	t.Run("it should filter clones with invalid metadata data", func(t *testing.T) {
		testCases := []struct {
			name      string
			stateData string
		}{
			{name: "absent clone and session", stateData: `{"c5bfsk0hmvjd7kau71jg": {}}`},
			{name: "absent session", stateData: `{"c5bfsk0hmvjd7kau71jg": {"clone": {"id": "c5bfsk0hmvjd7kau71jg"}}}`},
			{name: "absent clone", stateData: `{"c5bfsk0hmvjd7kau71jg": {"session": {"id": "1"}}}`},
			{name: "invalid session status", stateData: `{
  "c5bfsk0hmvjd7kau71jg": {
    "clone": {
      "id": "c5bfsk0hmvjd7kau71jg",
      "status": {
        "code": "FATAL"
      }
    },
    "session": {
      "id": "1"
    }
  }
}`},
		}

		prov, err := newProvisioner()
		assert.NoError(t, err)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				filepath, err := prepareStateFile(tc.stateData)
				assert.NoError(t, err)
				defer func() { _ = os.Remove(filepath) }()

				s := NewBase(nil, nil, prov, &telemetry.Agent{}, nil, nil)

				s.filterRunningClones(context.Background())
				assert.Equal(t, 0, s.lenClones())
			})
		}
	})
}
