# Security Policy

## Supported versions

Security fixes are applied to the current `main` branch and the latest published container image. Deploy the image by immutable digest and update after a successful CI publish.

## Reporting a vulnerability

Please use [GitHub private vulnerability reporting](https://github.com/arumes31/ts3news/security/advisories/new). Include the affected version or image digest, reproduction steps, impact, and any suggested mitigation. Do not disclose exploitable details in a public issue before a fix is available.

Authentication grants, TeamSpeak identities, database URLs, and API tokens are secrets. Redact them from reports, logs, screenshots, and test fixtures.
