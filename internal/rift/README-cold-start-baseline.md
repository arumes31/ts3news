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


## Input confirmation diagnostics

Open `/abyss/rift?riftInputDebug=1` for a bottom-left development overlay. It
reports the latest recognized-control-to-applied-response duration and rolling
p95 over at most120 samples. `window.RiftInputDiagnostics` contains the lifetime
sample count and bounded records with actions, total, queue and request milliseconds.
Records also split request time into `headers` (fetch until response headers),
`decode` (body transfer and JSON decoding), and `validation` (protocol hydration,
validation and baseline capture). `apply` is the remaining client work before
confirmation. These phases add up to the existing request and total measurements;
header wait includes network and server time, not server CPU time alone.
It is absent without the flag. No character or response payloads are retained.

Keyboard non-repeat presses, on-screen buttons, canvas mouse controls and controller
recognition callbacks mark the latest press of each action. Controller timing
starts at browser polling recognition, not physical button contact. A mark is
consumed when a step actually carries its movement direction, active combat flag
or selected skill ID. Multiple matching controls in one step form one sample,
starting at the earliest matching mark. Held repeats and idle polling add no samples.
Queue time includes input polling, ability buffering and request serialization;
request time ends after the response is parsed and protocol-validated. Total time
also includes applying the returned snapshot to the client. Pausing or resetting
inputs invalidates pending marks and in-flight measurements; failed requests do
not count. Buffered or blocked actions can remain pending until sent or replaced.

Confirmation means an input-bearing request was validated and applied. It does
not prove a hit, damage, successful cast, GPU presentation or physical-device input
latency. A response can legitimately report a blocked action. This diagnostic is
for development comparisons, adds overhead, and does not satisfy the physical
minimum-device timing gate by itself.


## Optional preview readiness

Mission-card CSS thumbnails and bestiary previews do not join `RiftRenderer.ready`.
Bestiary cards are constructed only when its details section opens. Preview URLs
reuse the versioned canvas atlas URLs; the combat atlases themselves remain required.
The optional-preview regression tags CSS-only consumers with a test query parameter
to isolate their requests from canvas loads, holds those requests indefinitely,
then verifies first play and live rendering before aborting them and confirming
pause/resume readiness without client errors. This verifies the dependency boundary,
not a download saving: all current combat atlases still load before first play.


## Pixel-exact compression experiment (2026-09-24)

The [lossless-art report](brawl-lossless-art-baseline.json) measures the22 atlas
files currently required by renderer.ready:18 Rift atlases plus4 shared Abyss
creature atlases. Source hashes identify the exact measured inputs. Pillow12.2.0
and libwebp1.6.0 encoded RGBA images with lossless=True,method=6,exact=True.
Every candidate was decoded and checked against every source RGBA byte, including
fully transparent pixels. Candidates were held in memory; source art and browser
URLs were not changed.

PNG total:44,399,088 bytes. Lossless WebP total:34,393,642 bytes. Saving:10,005,446
bytes(22.54%). Even the art-only transfer floor at200,000 B/s is171.97 seconds.
This excludes other requests, headers, latency, decoding and rendering and is not
a browser startup measurement. It decisively fails the3,000,000-byte readiness gate.

Format conversion alone therefore cannot satisfy startup readiness. The next
implementation must reduce the set of bytes required before play (for example,
scene-specific atlas loading), with explicit readiness before newly needed art is
shown. It must retain all monster/class/animation coverage and account for stage
transitions; simply enabling Start while required artwork is missing is insufficient.
The existing cold-start gate remains failed and must be remeasured after that work.

Reproduce without rewriting any artwork:

```powershell
python scripts/measure-brawl-lossless-art.py --output .tmp/brawl-lossless-art.json
```

Requires Pillow with WebP support. Asset discovery reads the renderer critical-key
list, template URLs and shared combat-art catalog list; mismatched source layout
fails rather than silently reporting a partial list. Encoder duration is local
experiment overhead, not browser decode time. The report's Pillow/libwebp versions
must accompany comparisons.


## Deferred legacy boss background (2026-09-25)

Ordinary startup now requires17 Rift atlases plus4 shared creature atlases.
The3,339,584-byte rift_boss_area.png is loaded only for a room2 snapshot without
region data. Initial saved-run loading and mutation response application await its
decode before exposing that scene. Concurrent preparation shares one request;
failure clears the promise so recovery can retry. Campaign region artwork is
unchanged, and the legacy background remains available.

The previous22-image compression report remains a historical experiment. Current
ordinary required-art bytes are41,059,504 (before other page resources). This saves
7.52% of the prior PNG atlas total, but still exceeds the3MB readiness target by a
large margin. No new throttled startup timing is claimed and the gate stays failed.
Browser checks confirm no boss-background request at campaign startup, saved-scene
readiness waiting, decoded reuse, failure retry, coalescing, and existing decode/
priority/progress/version behavior:11 passed. More scene-specific asset work remains.


## Selected hero atlas on demand (2026-09-25)

Ordinary startup now loads only the hero atlas required for the selected class
(or saved character build if different), rather than fetching both `rift_heroes_a.png`
and `rift_heroes_b.png` unconditionally. Critical base atlases drop to 15 (plus 4
shared creature atlases), with `prepareBuild` loading the appropriate hero sheet:
`heroesA` (classes 0-5) or `heroesB` (classes 6-11). If a saved expedition uses a
different class than the current build, both required hero atlases are loaded before
readiness is signaled.

This eliminates 2,608,798 to 2,704,155 bytes (~2.6-2.7 MB) from startup transfer,
bringing ordinary required artwork down to 38,355,349 - 38,450,706 bytes (a cumulative
~13.4% saving from the original 44,399,088 byte PNG baseline). The 3MB readiness gate
remains unmet and throttled startup timing remains unverified. 15 browser checks pass
across hero selection, saved-build readiness, deferred legacy boss art, atlas progress/
decode, priority, version reuse, and skill animation previews.


## Mission objective images on demand (2026-09-28)

[Objective artwork readiness](README-objective-art-loading.md) now removes seven
unused objective PNGs (8240354 bytes) from idle startup. Every objective required
by a mission's frozen tiers is decoded before its first scene, with current saved
actors included for legacy compatibility. Idle required PNG bytes are now
30114995–30210352; saved expeditions add only the needed objective images.
This remains far above the 3MB target. No new cold-start timing pass is claimed.

## Deferred legacy ordinary background (2026-09-28)

Modern regional startup no longer requests rift_area.png (3,149,894 bytes).
Seven base atlases plus four shared creature sheets and the selected hero remain
required. Saved regionless ordinary rooms prepare the original area background;
regionless boss rooms prepare the original boss background. Both paths wait for
decode, deduplicate concurrent loads, and clear rejected promises for retry.
The renderer now selects the active legacy scene explicitly instead of borrowing
the selected mission preview's region, and waits until its background is ready.

Browser checks verify absent requests in modern startup, blocked readiness until
legacy art arrives, actual legacy background draws, reuse, and retry for both
backgrounds. Decode/progress/priority/recovery checks pass with seven critical
atlases. The scenery oracle covers all 300 scenes, eight prop cells and 36 effect
cells, including source rectangles substituted by the offscreen cache; the
existing cached-versus-direct pixel oracle also passes. No source pixels changed.

The known file-size saving is not a startup timing result. The remaining required
art still exceeds the 3 MB budget; constrained-network readiness must be measured
on this candidate and remains an open performance gate.
