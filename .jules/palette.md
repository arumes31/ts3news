## 2024-07-27 - Inline Loading States for Go Templates
**Learning:** Adding loading states to inline event handlers in Go templates is easily achieved by passing `this` to the JavaScript function (e.g., `onclick="equip({{.InvID}}, this)"`). This allows direct access to the clicked element to add the `.btn-busy` class and disable it during async requests without needing complex DOM traversal.
**Action:** Always pass `this` when adding loading states to inline event handlers in list/grid views to easily target the specific button clicked.
