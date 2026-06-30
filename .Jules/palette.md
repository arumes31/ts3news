## 2024-05-24 - Missing global focus-visible states
**Learning:** The application lacked explicit `:focus-visible` styles, relying on browser defaults which were often insufficient or removed by CSS resets. This caused poor keyboard navigation accessibility.
**Action:** Implemented a global `:focus-visible` rule in `style.css` using the existing design system variable `--xp` (light blue) for outline and outline-offset, ensuring all interactive elements (links, buttons, inputs) have a clear and consistent focus indicator without needing per-component overrides.
