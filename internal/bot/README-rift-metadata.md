# Brawl public catalog and private initial state

GET /api/abyss/rift/metadata returns only class_names, class_options, rooms,
levels, bestiary and rarities. It accepts no player identifier and reads no user
state. The route is available only when Abyss is enabled. It derives content
from the same live canonical sources as the previous combined response.

The response has a SHA-256 content ETag and public, max-age=0, must-revalidate.
Matching strong/weak/list/wildcard If-None-Match returns304 with no body. Content
changes invalidate the tag. GET and HEAD are supported; mutation methods return405.
No private build, run, inventory, objective options or challenge is cacheable here.
The server still computes current metadata on revalidation; this change reduces
transfer size, not metadata generation CPU cost.

Private GET /api/abyss/rift remains authenticated and no-store. The new client
sends X-Rift-Metadata: separate to omit catalog fields, then fetches the public
endpoint without credentials using cache:no-cache. Only the six allowed fields
are merged before the existing complete-response validator runs. Abort/timeout,
session expiry, retry and generation ownership remain in the existing read flow.
A missing/invalid catalog prevents start and offers Retry loading. Legacy callers
without the header retain the original combined response. POST responses and
server persistence are unchanged; lean movement snapshots remain separate work.

Opt-in payload diagnostics count both decoded response bodies in a logical load
sample. They are decoded payload size, not proof of wire transfer on a cache hit.
The metadata fixture body measured289622 bytes; a matching revalidation body is0.
Unit checks cover the exact public allowlist,100 levels, strong/weak/list/wildcard
ETags, stale/content changes and rejection of mutations. Browser checks cover
split responses, omitted cookies,304 response, reload and failed-catalog retry
without starting an expedition. Implements0779/0780 after verification.
