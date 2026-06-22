## 2025-06-22 - Missing focus states in custom dark theme
**Learning:** Custom dark themes often rely on `border: none` and custom backgrounds for buttons, anchor tags, and input fields. This removes default browser focus outlines. Without explicit `:focus-visible` styles, keyboard accessibility becomes impossible for visually impaired users.
**Action:** Always ensure a global `:focus-visible` outline is added in projects with custom themes or CSS resets that suppress native outlines. Specifically in this application, using the `var(--xp)` color provides high contrast for focus outlines.
