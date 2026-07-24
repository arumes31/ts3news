## 2024-07-24 - Interactive loading states in inventory
**Learning:** Adding the `.btn-busy` CSS class alongside `disabled=true` is the standard pattern in this design system to give immediate visual feedback and prevent accidental double-submissions on asynchronous API operations (like equipping or vendoring items).
**Action:** Consistently pass `this` from HTML event handlers to JavaScript functions to easily target the clicked element, apply the `.btn-busy` class, and revert the state on failure.
