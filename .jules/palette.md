## 2024-05-24 - Loading States on Async Buttons
**Learning:** Adding `.btn-busy` (spinner via pseudo-element animation) and `disabled=true` on button click while waiting for async fetch requests prevents double submission and gives immediate visual feedback without layout shift. Passing `this` to inline `onclick` handlers in HTML allows easy targeting in vanilla JS without needing IDs.
**Action:** Always implement this simple pattern for destructive or async actions to immediately improve perceived performance and prevent accidental duplicates.
