## 2024-07-04 - Focus States for Accessibility

**Learning:** Missing `:focus-visible` styling is a common a11y issue in custom CSS. The `var(--accent)` color provides a good base for outlines without introducing new tokens. Using `box-shadow` instead of `outline` for form inputs provides a softer, more modern focus state while retaining visibility.
**Action:** Always check for interactive element focus states (buttons, links, inputs). Ensure they use `:focus-visible` to prevent annoying mouse-click outlines, while preserving keyboard navigation clarity.
