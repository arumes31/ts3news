## 2026-07-26 - Reusable Loading State Pattern
**Learning:** This app's design system uses a `.btn-busy` CSS class to show a loading spinner on interactive buttons, combined with `disabled = true` to prevent double submissions.
**Action:** When adding loading states to inline HTML event handlers (e.g., `onclick="equip({{.InvID}}, this)"`), always pass `this` to easily target the clicked element for dynamically adding `.btn-busy` and disabling it.
