# Touch target audit

On coarse-pointer devices, Brawl's buttons, selectors, input controls, disclosure
summaries and standalone links have a minimum 48 by 48 CSS-pixel target. Checkboxes
and radio buttons retain native glyph sizes inside full tappable labels with the
same minimum size. Inline prose links in the field guide are excluded from this
standalone-control rule. Fine-pointer desktop layouts keep their existing density.

The landscape battlefield remains compact during combat. At a checkpoint its
normal-flow banking panel expands the viewport container so both banking controls
fit before the touch movement pad. Fullscreen uses its existing layout rules.

`rift-touch-target-audit.spec.js` measures visible controls with every details
section expanded at 390x844, 844x390 and 1280x900 coarse-pointer viewports, then
opens the controls dialog and tests tapping a checkbox label away from its glyph.
It also measures boss/hazard/skill practice options and a live checkpoint/receipt
flow. No horizontal document overflow is allowed. Existing checkpoint tests cover
320x568, 390x844 and 844x390 banking and reward persistence; combat touch tests
cover gesture suppression without disabling normal outer-page selection.

This audit establishes target dimensions and the tested layouts/interactions; it
is not a claim of physical-device testing or of a complete accessibility audit.
