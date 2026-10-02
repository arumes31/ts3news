# Brawl display zoom audit

`tests/e2e/rift-native-zoom.spec.js` launches a separate, temporary Chromium profile
and a generated test-only extension. It uses the browser tabs API to apply native
125%, 150%, 200%, 300% and 400% zoom to a 1280x900 browser viewport. Each step checks
getZoom, devicePixelRatio and the resulting integer CSS viewport width. It does not
substitute viewport resizing, CSS zoom or pinch scaling for browser zoom.

The test starts from a confirmed checkpoint, selects the supported 125% HUD text
setting, scrolls controls into view and checks full bounds plus center-point hit
ownership. It opens/closes controls, resumes, banks/leaves and reaches the next
expedition action at each zoom. It records screenshots and technical measurements.
Only the loopback fixture and an isolated browser profile are used. The test
extension is never installed in a user's browser and nothing is uploaded.

High zoom exposed two layout problems: sticky wrapped site navigation obscured
Start in short viewports, and basic combat buttons stretched to the height of a
multirow skill grid. On Brawl pages with at most 480px viewport height the site
navigation scrolls with the document. At widths up to 600px basic and skill controls
occupy separate flex rows, allowing each control to fit the shortened viewport.

The older `rift-zoom-controls.spec.js` is explicitly a 320px reflow test. Its 200% HUD
property override now targets #rift-app, which owns that property, and asserts the
actual font size doubles. Previously changing the root property was overridden by
the component and did not prove enlarged text.

Scope: desktop Chromium native zoom and the tested controls/flows. This does not
claim every browser, OS magnifier or physical mobile pinch-zoom implementation.
Existing touch/reflow tests complement it without being described as native zoom.

Implementation references:
- [Playwright extension testing](https://playwright.dev/docs/chrome-extensions)
- [Chrome tabs zoom API](https://developer.chrome.com/docs/extensions/reference/api/tabs#method-setZoom)
