# Security policy

Security fixes target the current `main` branch. Deploy a digest from a successful
CI run with verified provenance and SBOM attestations. Older commits, experimental
branches, and historical `latest` images do not receive backported fixes.

## Reporting a vulnerability

Use [GitHub private vulnerability reporting](https://github.com/arumes31/ts3news/security/advisories/new)
to contact the maintainer, `@arumes31`. Include the affected commit/image digest,
reproduction steps, impact, and a minimal example with credentials and personal
data removed. Do not open a public issue containing an exploit, token, database
dump, TeamSpeak identity, or client profile. If private reporting is unavailable,
ask the maintainer to enable it without including vulnerability details publicly.

Disclose details publicly after coordinating a fix with the maintainer. No fixed
response-time commitment is offered. If a credential was exposed, revoke or rotate
it immediately; deleting it from the current tree does not remove it from history.

## Automated checks

- Every CI run on pushes and pull requests to `main` and `v2_test` runs
  `govulncheck`, a full-history redacted Gitleaks scan, source dependency scanning,
  and CodeQL for Go and JavaScript/TypeScript. The Security workflow repeats these
  checks every Monday at 06:23 UTC and supports manual runs. Scheduled workflows
  run on the default branch and can be delayed or disabled by GitHub.
- Anchore/Grype dependency and image scans fail on HIGH and CRITICAL findings,
  including findings without a fix. Scanner errors fail the job too. Fix findings
  or document a narrowly scoped, reviewed exception; do not disable the gate or
  introduce a blanket baseline to make an existing backlog green.
- CodeQL uploads source findings to the Security tab. Successful analysis alone
  does not mean no findings: configure a required code-scanning ruleset for HIGH
  and CRITICAL alerts. The publishing job depends on successful analysis; merge
  protection is what prevents merging code with outstanding CodeQL alerts.
- Publishing runs only for a push to `main`, after Go, browser, security, and
  candidate-image checks pass. The pushed GHCR digest is scanned again, inventoried
  as SPDX, and receives GitHub OIDC-signed provenance and SBOM attestations.
  A failed publish can leave a candidate in GHCR: a tag's existence is not approval.
- All external workflow actions use full commit SHAs with release comments.
  Dependabot checks Actions, Go, npm, and Docker dependencies weekly. Review SHA
  updates against upstream releases, including actions' transitive dependencies.
  Scanner versions and the golangci-lint tool version are explicit
  pins; Dependabot does not update version strings inside shell commands or action
  inputs, so maintainers must review those pins separately.

Coverage profiles, text/HTML reports, and scan reports are retained as workflow
artifacts. Secret scan findings are redacted in logs and are not uploaded as raw
reports. An SBOM is an inventory, not a guarantee of completeness; proprietary
TeamSpeak components and downloaded binaries still need upstream advisory review.

`.gitleaksignore` contains two exact historical fingerprints for the public
`Forge4Undo2Key` database-key prefix (`abyss_forge_undo2`), including its file move.
These are reviewed false positives, not credentials. No files or detection rules
are excluded; new occurrences still require review.

## Repository administration

Enable private vulnerability reporting, Dependabot security updates, secret
scanning, and push protection where available. Enable CodeQL advanced setup for
the checked-in workflow and remove conflicting default setup if configured.
Private repositories require the appropriate GitHub plan/features for CodeQL and
artifact attestations; do not silently skip them if unavailable.

Protect `main` and `v2_test` with required passing CI checks and code-scanning
results, code-owner review, stale-approval dismissal, and no force pushes. Ensure
`@arumes31` has write access so CODEOWNERS is effective. Require review of workflow,
scanner-policy, Docker, runtime, and migration changes. CODEOWNERS and workflow
files do not activate these server-side settings by themselves.

Keep default workflow token permissions read-only. Only the publisher gets package
write and OIDC/attestation permissions; only CodeQL jobs need security-event write.
Keep fork PR approval enabled and never execute PR code with `pull_request_target`
or production credentials. Grant the repository Actions token access to its GHCR
package; use read-only package credentials on deployment hosts.

See [deployment and migration guidance](DEPLOYMENT.md) for digest verification,
backups, and rollback.
