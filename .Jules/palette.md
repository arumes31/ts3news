## 2024-06-21 - Icon-Only Button Accessibility Pattern
**Learning:** Found multiple instances of icon-only `&times;` buttons across application panels (menus, settings, scoreboards) missing critical `aria-label`s. This pattern likely repeats in other template files where `&times;` or SVG icons are used without accompanying text.
**Action:** When adding new panels or reviewing templates, actively search for `.btn-close` or `&times;` usage and ensure a descriptive, context-specific `aria-label` (e.g., `Close settings`) is provided to maintain screen reader accessibility.
