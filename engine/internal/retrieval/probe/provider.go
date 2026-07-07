/*
2026 © Postgres.ai
*/

package probe

import "strings"

// Provider identifies a known managed-Postgres operator. The value is used as
// a stable key for telemetry payloads, the docker-image catalog in the UI, and
// the per-provider notes the Simple-mode preview renders.
type Provider string

// Known provider keys. Generic is the fallback when neither hostname nor
// extensions match a known operator.
const (
	ProviderRDS            Provider = "rds"
	ProviderAurora         Provider = "aurora"
	ProviderCloudSQL       Provider = "cloudsql"
	ProviderSupabase       Provider = "supabase"
	ProviderAzure          Provider = "azure"
	ProviderHeroku         Provider = "heroku"
	ProviderTimescaleCloud Provider = "timescale"
	ProviderGeneric        Provider = "generic"
)

// extension fingerprints (exact name match against pg_available_extensions.name)
const (
	extRDSTools        = "rds_tools"
	extAuroraStatUtils = "aurora_stat_utils"
	extPgGraphQL       = "pg_graphql"
	extSupabaseVault   = "supabase_vault"
	extCloudSQLIAM     = "cloudsql_iam"
)

// hostname suffix rules (case-insensitive match on the trailing portion of the host).
var hostnameRules = []struct {
	suffix   string
	provider Provider
}{
	{".rds.amazonaws.com", ProviderRDS},
	{".supabase.co", ProviderSupabase},
	{".supabase.com", ProviderSupabase},
	{".pooler.supabase.com", ProviderSupabase},
	{".postgres.database.azure.com", ProviderAzure},
	{".tsdb.cloud.timescale.com", ProviderTimescaleCloud},
}

// DetectProvider classifies a source by hostname suffix and available
// extensions. The available-extensions list is expected to be the result of
// `SELECT name FROM pg_available_extensions` on the source (no
// installed-version filter — Aurora-distinguishing extensions are typically
// available but not installed).
//
// Matching rules:
//   - hostname suffix wins unambiguously (RDS, Supabase, Azure, TimescaleCloud);
//   - on RDS hosts, the presence of `aurora_stat_utils` upgrades the result to
//     Aurora;
//   - when the hostname is generic or private, extension fingerprints fill in:
//     `cloudsql_iam` → CloudSQL, `pg_graphql` + `supabase_vault` → Supabase,
//     `rds_tools` alone (no Aurora marker) → RDS;
//   - Heroku has no distinctive extensions and no stable host suffix today, so
//     it falls through to Generic until a future revision adds a Heroku DSN
//     heuristic.
//
// Returns ProviderGeneric when no rule fires.
func DetectProvider(host string, availableExtensions []string) Provider {
	exts := make(map[string]struct{}, len(availableExtensions))
	for _, e := range availableExtensions {
		exts[e] = struct{}{}
	}

	lower := strings.ToLower(host)

	for _, rule := range hostnameRules {
		if !strings.HasSuffix(lower, rule.suffix) {
			continue
		}

		// RDS hostname can be Aurora — break the tie via extension marker.
		if rule.provider == ProviderRDS {
			if _, ok := exts[extAuroraStatUtils]; ok {
				return ProviderAurora
			}
		}

		return rule.provider
	}

	if _, ok := exts[extCloudSQLIAM]; ok {
		return ProviderCloudSQL
	}

	if _, vaultOK := exts[extSupabaseVault]; vaultOK {
		if _, gqlOK := exts[extPgGraphQL]; gqlOK {
			return ProviderSupabase
		}
	}

	if _, ok := exts[extAuroraStatUtils]; ok {
		return ProviderAurora
	}

	if _, ok := exts[extRDSTools]; ok {
		return ProviderRDS
	}

	return ProviderGeneric
}
