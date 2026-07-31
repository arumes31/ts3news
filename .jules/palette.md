## 2024-05-24 - Inline Handlers Need explicit 'this'
**Learning:** When using Go templates or vanilla HTML with inline `onclick` handlers, it's very easy to forget to pass `this` to the function. This makes it hard to quickly add visual loading indicators like `.btn-busy` or to disable the button to prevent double-submissions.
**Action:** Always ensure that inline event handlers pass `this` (e.g. `onclick="equip(id, this)"`), and that the Javascript functions accept the element and apply `.btn-busy` and `.disabled = true` immediately, restoring them on failure.
