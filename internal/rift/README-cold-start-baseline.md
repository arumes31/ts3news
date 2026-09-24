# Brawl constrained-network startup baseline

Measured 2026-09-24 against the isolated local fixture built with the startup-read
recovery change at `096aeddd7b1d46a267117267b0e65428533621e2`. No production
accounts, saves or inventory were used. This is a performance baseline, not a
release-readiness or acceptable-loading-time claim.

## Method

Used the [cold-start measurement command](../../scripts/README-brawl-cold-start.md)
against port 18097 with Chromium 153.0.8010.12 on Windows. Three sequential fresh
contexts, disabled HTTP cache, blocked service workers, 1280x900 viewport, en-US
locale, UTC time zone and no CPU throttle. Network emulation: 150ms latency,
200,000 bytes/sec download (1.6Mbps), 93,750 bytes/sec upload (0.75Mbps).

Readiness means the Start button is enabled after required artwork and initial
API data are available. It is distinct from document load or the first combat
frame. This is a synthetic local-server test, not a real-user mobile measurement.

## Results

| Sample | Ready | DOM content loaded | Transferred at readiness | Recovery |
| --- | ---: | ---: | ---: | --- |
| 1 | 230.418s | 10.538s | 45,949,975 bytes | One initial GET abort, recovered |
| 2 | 230.403s | 10.519s | 45,949,975 bytes | One initial GET abort, recovered |
| 3 | 230.379s | 10.490s | 45,949,975 bytes | One initial GET abort, recovered |

Median readiness: **230.403 seconds** (about 3 minutes 50 seconds). Each sample
completed 55 subresources by readiness, with 45,933,475 encoded body bytes across
the document and completed resources. Every sample reached 18/18 critical atlases;
the renderer readiness promise also awaited the shared combat atlases. The only
reported error in each sample was the recovered GET `/api/abyss/rift` abort.
No other request or page errors were reported.

Largest completed downloads in the first sample:

| Asset | Encoded body bytes |
| --- | ---: |
| `rift_boss_area.png` | 3,339,584 |
| `rift_regions.png` | 3,222,009 |
| `rift_area.png` | 3,149,894 |
| `rift_mobs.png` | 2,782,428 |
| `rift_platform_surface.png` | 2,771,164 |

The 22 required base/shared atlases total 44,399,088 source bytes. At the configured
download rate, that alone implies about 222 seconds of transfer. The measured
byte count and elapsed time are close to the connection's limit, supporting
initial download volume as the dominant bottleneck in this experiment.

## Startup failure found during measurement

Before the recovery change, increasing the measurement timeout from three to six
minutes did not produce readiness. A diagnostic run found Retry loading at
20.532 seconds, with 0/18 atlases ready and the initial API GET aborted after
10.006 seconds. Parallel atlas downloads prevented that bounded read from
completing. Once the page entered its recovery state, waiting could not fix it.

The fix retries one aborted initial GET after artwork finishes. It also preserves
GET body-abort identity, while leaving mutation response handling unchanged.
Focused regressions cover fetch/body aborts, persistent failure, truncated initial
responses, banking recovery and atlas progress. All six browser tests passed.

Recovery restores eventual usability on this connection. It does not solve the
large startup download. Prioritize reducing initially required asset bytes, then
repeat this same profile and compare readiness, byte totals and recovery frequency.

## Evidence and limits

Raw local reports are retained in `.tmp/brawl-cold-start-fixed-batch468.json`,
`.tmp/brawl-cold-start-diagnostic-batch467.json`, and the earlier timeout reports.
They contain resource paths/timings and synthetic-browser diagnostics, not player
records or credentials. This document preserves the key results for review.

The command records checkout revision, not independently attested server revision;
the fixture was rebuilt from the changed local sources before this run. Browser,
content, server, hardware and network changes can alter the results. Three samples
show consistency for this setup, not population-wide latency. In-flight requests
after readiness, image decode/frame performance, long-session memory and real
production database latency are separate measurements.

## In-session frame diagnostics

Open `/abyss/rift?riftFrameDebug=1` (or append `&riftFrameDebug=1` to an existing
query) for a development overlay at the bottom-right of the battlefield. It
reports mean and p95 completed-frame intervals and JavaScript render duration
in milliseconds over at most120 recent samples. Text updates at most twice per
second. `RiftRenderer.frameDiagnostics` exposes the bounded samples and lifetime
count for local profiling; diagnostics are null when the flag is absent.

Intervals include intentional30/15 FPS limits, browser scheduling delays and
render work. Render duration measures synchronous JavaScript canvas submission,
not GPU completion, compositor presentation or end-to-end input latency. Hidden
page intervals are excluded by resetting the interval origin on visibility loss.
Diagnostics add measurement overhead and are not enabled by default. Record
viewport, browser/device, selected FPS and scene alongside any reported values.
