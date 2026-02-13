package snapshot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDatabaseRenames(t *testing.T) {
	tests := []struct {
		name    string
		renames map[string]string
		connDB  string
		wantErr bool
	}{
		{name: "empty map", renames: map[string]string{}, connDB: "postgres", wantErr: false},
		{name: "valid renames", renames: map[string]string{"prod_db": "dblab_db"}, connDB: "postgres", wantErr: false},
		{name: "multiple valid renames", renames: map[string]string{"db1": "db1_new", "db2": "db2_new"}, connDB: "postgres", wantErr: false},
		{name: "rename matches connDB", renames: map[string]string{"postgres": "pg_renamed"}, connDB: "postgres", wantErr: true},
		{name: "one of multiple matches connDB", renames: map[string]string{"safe_db": "new_safe", "postgres": "renamed"}, connDB: "postgres", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDatabaseRenames(tt.renames, tt.connDB)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "connection database")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBuildRenameCommand(t *testing.T) {
	tests := []struct {
		name     string
		username string
		connDB   string
		oldName  string
		newName  string
		expected []string
	}{
		{
			name: "simple rename", username: "postgres", connDB: "postgres", oldName: "prod_db", newName: "dblab_db",
			expected: []string{"psql", "-U", "postgres", "-d", "postgres", "-XAtc", `ALTER DATABASE "prod_db" RENAME TO "dblab_db"`},
		},
		{
			name: "special characters in name", username: "admin", connDB: "management", oldName: "my-db", newName: "my_db",
			expected: []string{"psql", "-U", "admin", "-d", "management", "-XAtc", `ALTER DATABASE "my-db" RENAME TO "my_db"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRenameCommand(tt.username, tt.connDB, tt.oldName, tt.newName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
