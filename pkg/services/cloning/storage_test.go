package cloning

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/services/provision"
)

const (
	testingCloneState = `
{
  "c5bfsk0hmvjd7kau71jg": {
    "clone": {
      "id": "c5bfsk0hmvjd7kau71jg",
      "snapshot": {
        "id": "east5@snapshot_20211001112229",
        "createdAt": "2021-10-01 11:23:11 UTC",
        "dataStateAt": "2021-10-01 11:22:29 UTC"
      },
      "protected": false,
      "deleteAt": "",
      "createdAt": "2021-10-01 12:25:52 UTC",
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
			assert.NotEmpty(t, s.clones)
			assert.Contains(t, s.clones, "c5bfsk0hmvjd7kau71jg")
			assert.Equal(t, "east5@snapshot_20211001112229", s.clones["c5bfsk0hmvjd7kau71jg"].Clone.Snapshot.ID)
			assert.Equal(t, "east5", s.clones["c5bfsk0hmvjd7kau71jg"].Session.Pool)
			assert.Equal(t, uint(6003), s.clones["c5bfsk0hmvjd7kau71jg"].Session.Port)
		})
	})
}

func TestSavingSessionState(t *testing.T) {
	t.Run("it should save even if a clone list is empty", func(t *testing.T) {
		f, err := os.CreateTemp("", "dblab-clone-state-test-*.json")
		assert.NoError(t, err)
		defer func() { _ = os.Remove(f.Name()) }()

		s := NewBase(nil, nil, nil)
		err = s.saveClonesState(f.Name())
		assert.NoError(t, err)

		data, err := os.ReadFile(f.Name())
		assert.NoError(t, err)

		assert.Equal(t, "{}", string(data))
	})
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

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				filepath, err := prepareStateFile(tc.stateData)
				assert.NoError(t, err)
				defer func() { _ = os.Remove(filepath) }()

				s := NewBase(nil, &provision.Provisioner{}, nil)

				s.filterRunningClones(context.Background())
				assert.Equal(t, 0, len(s.clones))
			})
		}
	})
}
