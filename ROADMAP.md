# DBLab Engine 5 roadmap

Planned directions, not a delivery sequence. Platform roadmap: [platform-all/ROADMAP.md](https://gitlab.com/postgres-ai/platform-all/-/blob/main/ROADMAP.md).

## DBLab 5

- PostgresAI Cloud for DBLab Engine
- Installation in PostgresAI Cloud from CLI, with auto-config based on connection to source
- Installation instructions for AI agents (skill) for PostgresAI Cloud
- Full PostgresAI CLI/MCP for DBLab
- Continuous logical mode (logical replication, with DDL support)
- Data masking / PII obfuscation based on configs
- AI-based guessing of columns to obfuscate, for existing schema and new columns (optional manual review; GitOps)
- PG19 support
- Minor version control
- Skip-release major upgrades
- Full glibc version control
- Optimization loops in PostgresAI (via Joe MCP)
- Improved local experience (MacOS including Silicon, Ubuntu)
- pull/push
