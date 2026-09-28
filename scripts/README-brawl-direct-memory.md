# Direct-browser memory comparison driver

The direct CDP transport and Chromium launcher are groundwork for comparing
network-inspector overhead under the same synthetic Brawl gameplay workload.
They do not provide a completed memory comparison or close ledger0798.

`brawl-direct-cdp.cjs` routes commands and target-session notifications over one
WebSocket. It enables no CDP domains implicitly. Callers must explicitly enable
Runtime/Page and choose whether Network inspection belongs in their sample.
Timeout, protocol-error and disconnect handling reject pending commands; listeners
can be removed after heap snapshot streaming. Three Node tests cover routing,
error handling and timeouts.

`brawl-direct-chromium.cjs` starts the installed Playwright Chromium executable
directly, using an isolated profile under the caller's ignored output directory.
It binds remote debugging to loopback, launches headless with windows hidden,
and closes only its own browser. It does not connect to the user's open browser.
Profiles and diagnostic stderr stay local and must not enter public evidence.

The September28 launch smoke used Chrome153.0.8010.12, created a separate target,
enabled Runtime, evaluated2+2 and confirmed visible document state, then closed
the browser. No Network.enable command was sent. This proves basic transport and
launch, not campaign execution, comparable rendering or stable process memory.
Next reuse the complete three-tier keyboard navigator and checkpoint protocol
for paired Network-enabled/disabled captures with identical launch settings.
Record exceptions and HTTP failures independently in the disabled variant.
