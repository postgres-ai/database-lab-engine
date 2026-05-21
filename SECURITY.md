# Database Lab Engine security guidelines

## Reporting vulnerabilities
We're extremely grateful for security researchers and users that report vulnerabilities directly to Postgres.ai and the Database Lab Engine Community. All reports are thoroughly investigated by a set of community volunteers and the Postgres.ai team.

To report a security issue, please email us at security@postgres.ai with all the details, attaching all necessary information.

### When should I report a vulnerability?
- You think you have discovered a potential security vulnerability in the Database Lab Engine or related components.
- You are unsure how a vulnerability affects the Database Lab Engine.
- You think you discovered a vulnerability in another project that Database Lab Engine depends on (e.g., ZFS, Docker, etc).
- You want to report any other security risk that could potentially harm Database Lab Engine users.

### When should I NOT report a vulnerability?
- You're helping tune Database Lab Engine for security, perform some planned experiments coordinated with the maintainers.
- Your issue is not security related.

## Security Vulnerability Response
Each report is acknowledged and analyzed by the project's maintainers and the security team within 3 working days.

The reporter will be kept updated at every stage of the issue's analysis and resolution (triage -> fix -> release).

## Public Disclosure Timing
A public disclosure date is negotiated by the Postgres.ai security team and the bug submitter. We prefer to fully disclose the bug as soon as possible once user mitigation is available. It is reasonable to delay disclosure when the bug or the fix is not yet fully understood, the solution is not well-tested, or for vendor coordination. The timeframe for disclosure is from immediate (especially if it's already publicly known) to a few weeks. We expect the time-frame between a report to public disclosure to typically be in the order of 7 days. The Database Lab Engine maintainers and the security team will take the final call on setting a disclosure date.

## Known unfixed vulnerabilities
Some third-party CVEs cannot yet be patched in DBLab Engine for reasons outside of the project's control (no upstream fix released, major-version migration required, or embedded in base-image internals). They are tracked here and resolved as upstream fixes become available.

| Component | CVE(s) | Notes |
|-----------|--------|-------|
| `github.com/docker/docker` v28.5.2+incompatible (Go module) | [CVE-2026-34040](https://nvd.nist.gov/vuln/detail/CVE-2026-34040) | No v29 tag is published for this module. Upstream has moved to `github.com/moby/moby/v2`, still in beta at the time of writing. Will be resolved once a stable v29 release or the v2 migration is available. |
| `docker:29.x` base-image embedded binaries (`containerd`, `ctr`, `dockerd`, `compose`, `buildx`) | Multiple — see the [Docker Hub `docker` image advisories](https://hub.docker.com/_/docker/tags) and upstream tracker pages for [containerd](https://github.com/containerd/containerd/security/advisories) and [buildx](https://github.com/docker/buildx/security/advisories) | Depends on Docker Inc. rebuilding `docker:29.x` with updated internals. Tracked and refreshed together with each base-image bump; images are pinned by digest so any rebuild lands via an explicit commit. |

Operators should subscribe to upstream advisories for the affected components and re-deploy once DBLab Engine images built against patched versions are published.


*This document has been inspired by and adapted from [https://github.com/hasura/graphql-engine/blob/master/SECURITY.md](https://github.com/hasura/graphql-engine/blob/master/SECURITY.md).*
