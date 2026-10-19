# AGENTS.md

## Non-trivial changes (PostgresAI team)

For any non-trivial change (a new feature, a behaviour change, or anything spanning several repos),
use the `/feature-loop:feature-loop` skill (`/feature-loop` if you copied the file by hand). It
passes every story end to end before review and caps REV at 3 rounds (about 1 hour per repo), so it
replaces a separate open-ended REV loop. Skip it for typos, docs-only edits, routine version bumps
with no expected behaviour change, and small fixes covered by one test.

The skill lives in [postgres-ai/air](https://gitlab.com/postgres-ai/air/-/tree/main/skills)
(team access only). Install it with:

```bash
claude plugin install feature-loop --marketplace git@gitlab.com:postgres-ai/air.git
```

If the skill isn't installed, carry on without it.
