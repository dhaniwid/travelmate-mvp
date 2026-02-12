---
trigger: always_on
---

# MIRU ENGINEERING STANDARDS (Global Rules)

## 1. Tech Stack Constraints
- **Backend:** Go (Golang) with Chi router. Strict idiomatic Go.
- **Frontend:** Next.js 14+ (App Router), TypeScript, Tailwind CSS, Lucide React icons.
- **Database:** PostgreSQL with raw SQL migrations (or Prisma if specified).
- **Styling:** Use `clsx` and `tailwind-merge` for class manipulation. DO NOT write custom CSS unless absolutely necessary.

## 2. Coding Philosophy (The "Atlas" Standard)
- **Error Handling:** Never ignore errors in Go (`_`). Always handle or wrap them (`fmt.Errorf("context: %w", err)`).
- **Type Safety:** No `any` in TypeScript. Define strict interfaces in `src/types/`.
- **Modularity:** Frontend components must be small. If a component exceeds 200 lines, suggest splitting it.
- **Async Logic:** Prefer Go Goroutines for heavy tasks (like AI calls). Always use `context.Context` to manage timeouts.

## 3. UX/UI Guidelines (The "Kai" Standard)
- **Mobile-First:** Always assume the user is on a phone. Tap targets must be >44px.
- **Loading States:** Never leave the user hanging. Always implement Skeleton Loaders or Spinner states for async data.
- **Feedback:** Every action (Save, Delete, Generate) must have a Toast notification (Success/Error).
- **Aesthetics:** Use "Glassmorphism" (bg-white/10 backdrop-blur) for overlays. Round corners (`rounded-2xl` or `rounded-3xl`) are mandatory for cards.

## 4. Project Context
- We are building "Miru" (formerly TravelMate), an AI travel planner.
- Core feature: "Progressive Generation" (Fast skeleton -> Background enrichment).
- Routes: Backend runs on port 8080 (or 8889), Frontend on 3000.