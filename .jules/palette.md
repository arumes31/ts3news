## 2024-07-17 - Added btn-busy to inventory actions
**Learning:** The `.btn-busy` CSS class handles adding a loading spinner and hiding button text without changing layout. The class needs `btn.disabled = true` to fully prevent interactions.
**Action:** Always add `.btn-busy` and `.disabled = true` on interactive buttons making asynchronous API requests across the portal to provide visual feedback and prevent double-clicks.
