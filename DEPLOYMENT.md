# GHCR deployment and migration

Production bot and Idely containers use the same immutable reference:
`ghcr.io/arumes31/ts3news@sha256:<64 hexadecimal characters>`.
The checked-in `docker-compose.ghcr.yml` requires `TS3NEWS_IMAGE_DIGEST`; it has no
`latest` fallback. Images currently support `linux/amd64` because the official
TeamSpeak client is an amd64 binary.

## Select and verify an image

Choose a successful **CI** run for a reviewed commit on `main`. Download its
`image-supply-chain` artifact and read `image-reference.txt`. The job summary also
shows the reference after scanning and both signing steps succeed. The artifact
contains the SPDX SBOM and the exact-digest scan report; coverage is in the separate
`go-coverage` artifact. Keep this evidence with the deployment record before
GitHub's artifact retention expires.

The `sha-<full Git commit>` tag is a discovery aid, not immutable: a workflow rerun
can rebuild the same commit with newer base packages. Always use the recorded
digest for reproducible deployments. CI automatically updates
`ghcr.io/arumes31/ts3news:latest` to the same verified digest after registry scans,
SBOM checks, both signing steps and evidence upload succeed. Runs whose commit is
no longer the current `main` head skip this promotion, so an old rerun cannot roll
the tag back. Updating the tag does not restart existing containers; tag-based
deployments must pull and recreate their containers to use the new image.
Never deploy a candidate just because its `sha-<commit>` tag exists: registry
scanning happens after push and a failed run may leave it behind.

With a current GitHub CLI, authenticate to GitHub and GHCR as needed, then verify
both attestations. Replace the example values with the selected run's digest and
full Git commit SHA:

```bash
export TS3NEWS_IMAGE_DIGEST='sha256:<64 hexadecimal characters>'
export SOURCE_COMMIT='<40-character Git commit SHA>'
IMAGE="ghcr.io/arumes31/ts3news@$TS3NEWS_IMAGE_DIGEST"

gh attestation verify "oci://$IMAGE" \
  --repo arumes31/ts3news \
  --signer-workflow arumes31/ts3news/.github/workflows/ci.yml \
  --source-ref refs/heads/main --source-digest "$SOURCE_COMMIT" \
  --predicate-type https://slsa.dev/provenance/v1

gh attestation verify "oci://$IMAGE" \
  --repo arumes31/ts3news \
  --signer-workflow arumes31/ts3news/.github/workflows/ci.yml \
  --source-ref refs/heads/main --source-digest "$SOURCE_COMMIT" \
  --predicate-type https://spdx.dev/Document/v2.3
```

Stop if either command fails. Verification binds the digest to this repository,
workflow, branch, and source commit, rather than accepting any valid signature.
BuildKit also embeds provenance metadata, but that metadata alone does not
replace verification of the OIDC-signed attestations. Attestations prove identity
and origin; inspect the successful run and scan evidence for the vulnerability
gate result. They do not assert that an image will remain vulnerability-free.

For a private package, log Docker into GHCR using a credential with `read:packages`
and any required organization authorization. Use `docker login ghcr.io --username
YOUR_USER --password-stdin` with a secret manager or secure prompt supplying stdin.
Do not place tokens in Compose, shell history, or the repository.

## Migrate an existing deployment

1. Record the running image digest, source commit, Compose project name, service
   configuration, mounts, and database schema version. Disable tag-based automatic
   updaters for these services. Retain the previous image and attestations in GHCR
   so registry cleanup cannot remove your rollback target.
2. Compare the old and new commits' `internal/db/migrations/` and
   `internal/db/gold_reset.go`. Schema up-migrations are embedded in the binary and
   run at startup. Startup also applies versioned gold-economy resets; this can
   change player data even without a schema change. Rehearse the upgrade on a
   restored copy of the production database with external bot activity disabled.
3. Stop bot and Idely writes, back up PostgreSQL, `config.env`, persistent client
   identities/profiles, and TS3AudioBot data. Protect backups as secrets and test a
   restore. Preserve the existing PostgreSQL 15 image and database storage layout.
4. Add the verified `TS3NEWS_IMAGE_DIGEST=sha256:...` to the project's `.env`, or
   export it in the deployment shell. A service's `env_file: config.env` supplies
   container variables; it does **not** supply Compose interpolation. If you keep
   deployment values in `config.env`, explicitly pass `--env-file config.env` on
   every Compose command instead. Preserve the existing Compose project name and
   mount paths when switching files.
5. Pull the digest and start the bot first, then Idely after migrations and portal
   checks succeed. Check logs, TeamSpeak connectivity, and the web portal; a
   running container alone is not proof of a healthy application.

For an existing installation already using `docker-compose.ghcr.yml`, a Bash
example is below. Verify the image first and adapt the backup destination to your
protected backup storage. These commands use `.env` for interpolation.

```bash
set -euo pipefail
umask 077
docker compose -f docker-compose.ghcr.yml stop ts3-bot ts3-idely
docker compose -f docker-compose.ghcr.yml exec -T db \
  sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' \
  > "ts3news-before-upgrade-$(date -u +%Y%m%dT%H%M%SZ).dump"
docker compose -f docker-compose.ghcr.yml pull ts3-bot ts3-idely
docker compose -f docker-compose.ghcr.yml up -d --no-deps ts3-bot
docker compose -f docker-compose.ghcr.yml logs --tail=100 ts3-bot
# After verifying startup, migrations, TeamSpeak and portal behavior:
docker compose -f docker-compose.ghcr.yml up -d --no-deps ts3-idely
```

When migrating from the README's older minimal Compose example, its
`postgres_data` named volume is different from the GHCR file's `./data/postgres`
bind mount. Preserve your current database mount or perform a tested dump/restore;
blindly replacing that Compose file would start an empty database. Keep custom
ports, environment values, identities, and sidecar configuration too. For a new
installation, configure `config.env`, set the verified digest in `.env`, then run
`docker compose -f docker-compose.ghcr.yml up -d`.

Do not combine the GHCR file with the source-build Compose file: use the explicit
`-f docker-compose.ghcr.yml` commands to avoid inheriting `build:` or local override
settings. The default `docker-compose.yml` remains the source-build option.
The image uses Go 1.27.1 and Debian 13 (trixie), with UPX packing removed so image
scanners can inventory Go modules. It introduces no application-schema migration,
and retains the existing PostgreSQL 15 container and storage layout.

## Rollback and failed migrations

If the old binary is compatible with the upgraded schema and data, restore the
previous verified digest in `.env`, pull it, and recreate both bot services with
`up -d --no-deps ts3-bot ts3-idely`. Verify startup and portal behavior again.
Changing an image digest does not undo a database migration or economy reset.

For an incompatible change, stop all application writers and restore the tested
pre-upgrade backup into an isolated replacement database, then start the old image
against it. Account for writes since the backup before cutover. Do not blindly run
down-migrations, reset a dirty flag, or delete database volumes. The current
migrator contains a special dirty-version-39 recovery; that is not a general
recovery procedure. Investigate the exact failed SQL and database state first.

PostgreSQL, BusyBox, and TS3AudioBot are independent upstream images in this Compose
stack. BusyBox and TS3AudioBot digests are pinned independently; PostgreSQL remains
on its existing `15-alpine` tag. None is covered by the bot's attestations.
Review future digest changes and plan database major upgrades independently. Docker
base images are also digest-pinned; apt packages and downloaded TeamSpeak components
can still change between builds, so a Git commit cannot replace an image digest.

References: [GitHub artifact attestations](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations),
[verification CLI options](https://cli.github.com/manual/gh_attestation_verify),
[Docker Compose interpolation](https://docs.docker.com/compose/how-tos/environment-variables/variable-interpolation/).

## Dependency refresh (September 2026)

- Go 1.27.1, Node 26.8.1, Playwright 1.63.0, Pillow 12.3.0, and `x/image` 0.45.0.
- Debian 13 (trixie) image bases are pinned by digest. PostgreSQL remains unchanged
  at `postgres:15-alpine`, mounted at `/var/lib/postgresql/data`.
- BusyBox is pinned to the latest tagged release, 1.38.0; upstream labels this
  release line unstable. Validate the one-shot initialization service before use.
- TeamSpeak 3.6.2 remains the latest TS3 client with the required ClientQuery
  interface. `ancieque/ts3audiobot:latest` has no newer published image; its existing
  latest digest is now pinned. These older upstream components still need security
  review and are not guaranteed vulnerability-free by an upgrade or a signature.
