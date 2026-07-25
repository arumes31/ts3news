## 2023-10-24 - Inventory Loading States
**Learning:** For inline HTML event handlers (e.g., `onclick="equip(...)"`), passing `this` and applying the `.btn-busy` class while setting `disabled = true` is a reliable pattern in this app to prevent double submissions and provide essential loading feedback for async operations.
**Action:** Consistently pass `this` from inline handlers and utilize `btn.disabled = true; btn.classList.add('btn-busy');` to give visual cues to the user when waiting for API responses.
