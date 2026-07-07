/*
2026 © Postgres.ai
*/

package probe

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolvePreloadLibs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want string
	}{
		{name: "empty input gets pg_stat_statements", in: nil, want: "pg_stat_statements"},
		{name: "missing pg_stat_statements is injected", in: []string{"pgaudit", "pg_cron"}, want: "pg_cron,pg_stat_statements,pgaudit"},
		{name: "single entry containing the lib stays as-is", in: []string{"pg_stat_statements"}, want: "pg_stat_statements"},
		{name: "duplicates removed", in: []string{"pgaudit", "pgaudit", "pg_stat_statements"}, want: "pg_stat_statements,pgaudit"},
		{name: "alphabetical order", in: []string{"zztop", "aaaa", "mmm"}, want: "aaaa,mmm,pg_stat_statements,zztop"},
		{name: "whitespace trimmed", in: []string{" pgaudit ", "pg_cron"}, want: "pg_cron,pg_stat_statements,pgaudit"},
		{name: "blank entries dropped", in: []string{"", "  ", "pgaudit"}, want: "pg_stat_statements,pgaudit"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, ResolvePreloadLibs(tc.in))
		})
	}
}
