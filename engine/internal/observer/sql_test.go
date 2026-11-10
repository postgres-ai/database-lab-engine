/*
2020 © Postgres.ai
*/

package observer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestBuildConnectionString(t *testing.T) {
	tests := []struct {
		name      string
		clone     *models.Clone
		socketDir string
		want      string
	}{
		{
			name: "simple values",
			clone: &models.Clone{
				DB: models.Database{
					Port:     "5432",
					Username: "admin",
					DBName:   "testdb",
				},
			},
			socketDir: "/var/run/postgresql",
			want:      "host=/var/run/postgresql port=5432 user='admin' database='testdb' application_name='observer'",
		},
		{
			name: "username with single quotes",
			clone: &models.Clone{
				DB: models.Database{
					Port:     "5432",
					Username: "user'name",
					DBName:   "mydb",
				},
			},
			socketDir: "/tmp",
			want:      "host=/tmp port=5432 user='user\\'name' database='mydb' application_name='observer'",
		},
		{
			name: "dbname with backslash",
			clone: &models.Clone{
				DB: models.Database{
					Port:     "5432",
					Username: "user",
					DBName:   `db\name`,
				},
			},
			socketDir: "/tmp",
			want:      `host=/tmp port=5432 user='user' database='db\\name' application_name='observer'`,
		},
		{
			name: "special chars in both username and dbname",
			clone: &models.Clone{
				DB: models.Database{
					Port:     "6432",
					Username: `o'bri\en`,
					DBName:   `te'st\db`,
				},
			},
			socketDir: "/run/pg",
			want:      `host=/run/pg port=6432 user='o\'bri\\en' database='te\'st\\db' application_name='observer'`,
		},
		{
			name: "empty dbname falls back to default",
			clone: &models.Clone{
				DB: models.Database{
					Port:     "5432",
					Username: "user",
					DBName:   "",
				},
			},
			socketDir: "/tmp",
			want:      "host=/tmp port=5432 user='user' database='postgres' application_name='observer'",
		},
		{
			name: "spaces in username",
			clone: &models.Clone{
				DB: models.Database{
					Port:     "5432",
					Username: "my user",
					DBName:   "my db",
				},
			},
			socketDir: "/tmp",
			want:      "host=/tmp port=5432 user='my user' database='my db' application_name='observer'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildConnectionString(tt.clone, tt.socketDir)
			assert.Equal(t, tt.want, got)
		})
	}
}
