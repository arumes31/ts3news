## 2024-06-23 - Focus States and Base Accessibility
**Learning:** By default, base web applications may lack uniform focus-visible states across all elements. A missing global rule makes keyboard navigation entirely invisible for custom UI components, which significantly decreases accessibility.
**Action:** Always ensure a base `:focus-visible` rule is declared in the root stylesheet, taking advantage of existing design system variables (like `var(--accent)`).
