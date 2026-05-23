---
name: web-frontend
description: Frontend for knowledge-db in web/ (React 19, TypeScript, Vite, Tailwind). Use when editing web/src components, pages, hooks, or API client code.
---

# Frontend — web/

Stack:

- React 19, TypeScript, Vite
- `react-router-dom` — SPA routing
- Tailwind CSS 4 + Radix UI primitives + shadcn-style components
- `next-themes` for light/dark

Goal: dense, practical UI for managing a personal knowledge base — not a marketing site.

## Layout

```text
web/src/
├── main.tsx          # Entry, BrowserRouter
├── App.tsx           # Routes shell
├── pages/            # Add, Search, Node, Chat, Login, …
├── components/       # Shared UI
├── hooks/            # Data hooks
├── services/api.ts   # HTTP client
├── lib/              # Utilities (type-styles, headings, …)
└── types/            # API types
```

## Principles

- Functional components and hooks
- Fetch data in hooks/services, not scattered `useEffect` in presentational components
- Explicit `loading` / `error` / `success` for async UI
- Centralize API calls in `services/api.ts` (`VITE_API_URL`, default localhost)

## TypeScript

- Prefer precise interfaces over `any`
- API types in `types/` — **snake_case** field names matching backend JSON

## Routing

- Declarative routes in `App.tsx`
- Use `MemoryRouter` in tests — see [web-frontend-tests](../web-frontend-tests/SKILL.md)

## Styling

- Use Tailwind utility classes and shared tokens
- Node type colors: `web/src/lib/type-styles.ts` — do not duplicate badge/button classes (see `.cursor/rules/web-type-colors.mdc`)
- Keep UI compact: tables, filters, and forms should work on laptop-width viewports; avoid decorative chrome

## Forms

- Controlled inputs, labels, accessible errors — see [ux-form-practices](../ux-form-practices/SKILL.md)

## Commands

```bash
cd web && npm run dev
cd web && npm run build
cd web && npm run test
cd web && npm run lint
# or from repo root: task web:dev, task web:build, task web:test, task web:lint
```

## Checklist

- [ ] API changes reflected in `services/api.ts` and types
- [ ] Async states visible to the user
- [ ] Type colors via `type-styles.ts` when showing node types
- [ ] `npm run build` and `npm run test` when behavior changes
