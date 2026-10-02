# Brawl asset availability check

From the repository root, with a running target server:

```powershell
node scripts/check-brawl-assets.cjs http://127.0.0.1:18096 > .tmp/brawl-assets.json
```

Use the target's actual HTTP(S) origin. The example port is a local fixture, not
a production default. Requires Node with built-in fetch; verified with Node
24.19.0. No browser, game start, account cookie or extra package is needed.

The checker discovers fixed `/static/` references from this checkout's Brawl
page and shared page shell, then follows references in JavaScript and CSS. It
also includes the three shared class atlases whose filenames are assembled at
runtime. Local dependencies must exist and be nonempty. Conditional shell
references are included even when their section is not rendered in Brawl, so the
list is a conservative superset of the core Brawl page dependencies.

It sends four concurrent HEAD requests at most, each with a five-second timeout
and Cache-Control: no-cache. Redirects are not followed. Success requires HTTP
200, an expected MIME type and, when Content-Length is supplied, a nonzero length.
This catches missing files, login redirects and HTML fallback routes masquerading
as successful assets without downloading atlas bodies. The command does not start
or restart a server and never sends game actions.

Standard output is JSON with `checked`, `failed` and ordered per-asset results.
Standard error gives a compact summary. Exit codes:

- 0: every discovered asset is available.
- 1: one or more HTTP checks failed; all results are still reported.
- 2: invalid command options or local manifest discovery failure.

A server that does not support HEAD will fail this check; use its response details
to distinguish unsupported methods from missing files. Supply only an origin,
without credentials, path, query or fragment. Static paths resolve from `/static/`.

This checks availability against the checkout's dependency set, not deployed
revision equality. It does not verify image decoding, sprite frame bounds,
JavaScript execution, asset byte integrity, every dynamically generated catalog
portrait, or the appearance of the game. Browser and embedded-asset tests remain
necessary. Update the explicit runtime-generated class atlas list if the shared
art selector introduces another such atlas family.

```powershell
node --test scripts/check-brawl-assets.test.cjs
```

Tests cover dependency discovery, missing local files, request concurrency,
missing/redirected assets, wrong MIME, known-empty responses and request failure.
