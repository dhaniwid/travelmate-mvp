---
description: 
---

## 5. Debugging Protocol
- If a user reports a "404", ALWAYS check `routes.go` first.
- If a user reports "Hydration Error", check for nested HTML tags (e.g., `<div>` inside `<p>`, or `<button>` inside `<button>`).
- If adding a new env variable, ALWAYS remind the user to update `.env` and Railway variables.