## 2024-05-19 - Global Keyboard Focus Indicators
**Learning:** The custom dark theme lacks any `:focus` or `:focus-visible` styling for interactive elements like links, buttons, and inputs, making keyboard navigation nearly impossible as users cannot see what element is active.
**Action:** Add a global `:focus-visible` rule using the existing `--accent` CSS variable to all interactive elements to ensure clear visibility for keyboard users without impacting mouse interactions.
