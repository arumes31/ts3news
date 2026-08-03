## 2024-05-24 - Async Interaction Loading States
**Learning:** Implementing visual loading feedback directly into inline event handlers (passing 'this' element to disable it and attach a busy class) prevents double-submissions seamlessly within this template-driven architecture.
**Action:** Add '.btn-busy' and disable the button immediately on async invocation; remember to revert the state on failure.
