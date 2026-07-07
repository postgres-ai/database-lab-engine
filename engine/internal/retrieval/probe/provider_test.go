/*
2026 © Postgres.ai
*/

package probe

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		name string
		host string
		exts []string
		want Provider
	}{
		{name: "rds hostname only", host: "myinst.abc.us-east-1.rds.amazonaws.com", exts: nil, want: ProviderRDS},
		{name: "rds hostname uppercase", host: "MYINST.ABC.US-EAST-1.RDS.AMAZONAWS.COM", exts: nil, want: ProviderRDS},
		{name: "aurora hostname + ext marker", host: "myinst.abc.us-east-1.rds.amazonaws.com", exts: []string{"aurora_stat_utils", "rds_tools"}, want: ProviderAurora},
		{name: "rds hostname with rds_tools only", host: "myinst.abc.us-east-1.rds.amazonaws.com", exts: []string{"rds_tools"}, want: ProviderRDS},
		{name: "supabase hostname", host: "db.abc.supabase.co", exts: nil, want: ProviderSupabase},
		{name: "supabase pooler", host: "aws-0-us-east-1.pooler.supabase.com", exts: nil, want: ProviderSupabase},
		{name: "azure hostname", host: "myinst.postgres.database.azure.com", exts: nil, want: ProviderAzure},
		{name: "timescale cloud hostname", host: "abc.def.tsdb.cloud.timescale.com", exts: nil, want: ProviderTimescaleCloud},
		{name: "cloudsql via extension only", host: "10.20.30.40", exts: []string{"cloudsql_iam"}, want: ProviderCloudSQL},
		{name: "supabase via extensions only", host: "10.20.30.40", exts: []string{"pg_graphql", "supabase_vault"}, want: ProviderSupabase},
		{name: "supabase vault alone is not enough", host: "10.20.30.40", exts: []string{"supabase_vault"}, want: ProviderGeneric},
		{name: "rds via extension only", host: "10.20.30.40", exts: []string{"rds_tools"}, want: ProviderRDS},
		{name: "aurora via extension only", host: "10.20.30.40", exts: []string{"aurora_stat_utils"}, want: ProviderAurora},
		{name: "hostname wins over conflicting extension", host: "db.abc.supabase.co", exts: []string{"rds_tools"}, want: ProviderSupabase},
		{name: "unknown hostname, unknown extensions", host: "192.168.1.5", exts: []string{"pgaudit", "pg_cron"}, want: ProviderGeneric},
		{name: "empty host empty extensions", host: "", exts: nil, want: ProviderGeneric},
		{name: "heroku hostname falls through to generic", host: "ec2-50-50-50-50.compute-1.amazonaws.com", exts: nil, want: ProviderGeneric},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DetectProvider(tc.host, tc.exts)
			require.Equal(t, tc.want, got)
		})
	}
}
