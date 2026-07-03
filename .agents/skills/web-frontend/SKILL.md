---
name: web-frontend
description: React 19 + Vite + Tailwind frontend conventions — design-system usage, buttons, status chips, layout utilities, page/loading/error states.
---

# Modern React frontend (React 19 + Vite + Tailwind)

## Design system (mandatory)

Before changing UI, read the project's `design-system/MASTER.md` and match `design-system/preview.html`. Do not invent ad-hoc button or panel styles.

### Buttons

Shared styles live in the design-system stylesheet (e.g. `design-system/admin.css`). Always include the base class:

| Variant | `className` |
| ------- | ----------- |
| Primary | `button` |
| Secondary | `button secondary` |
| Danger | `button danger` |

```tsx
// correct
<button type="submit" className="button">Save</button>
<button type="button" className="button secondary">Cancel</button>

// wrong — no .secondary rule exists without .button
<button type="button" className="secondary">Cancel</button>
```

`<Link>` elements styled as buttons use the same classes. Filter pills use `filter-pill`; theme controls use `theme-button`.

### List queues

Use a consistent list pattern: `panel queue-panel` → filter bar → `entry-list` → `entry-card` rows with `chips` for metadata. Avoid bare `panel` + inline padding for list rows.

### Status chips

Use dedicated status chip components for entity state — not unstyled `chip` text. Each entity type may have its own chip component; keep styling in shared CSS.

### Layout utilities

Use shared utility classes (`section-heading`, `panel-form`, `inline-empty`, `toolbar`, `actions`, …) from the project's utilities stylesheet instead of inline `style={{}}`.

### Page states

`LoadingState` / `EmptyState` use `.loading` / `.empty` (large padding). Inside panels use `.inline-loading` / `.inline-empty`.

Every async view should handle loading, empty, error, and retry states.

### Stylesheets

| File | Scope |
| ---- | ----- |
| `design-system/admin.css` (or equivalent) | Core shell, buttons, tables |
| `src/utilities.css` | Shared layout helpers |
| Feature-specific CSS | Forms, domain-specific chips and panels |

Run `npm test` in the frontend package after UI changes — design-system tests guard class names where present.

## Conventions

- Routes defined in the app router (`App.tsx` or route module)
- API client module mirrors backend snake_case in request/response types
- Build: follow the project's task/Makefile targets to compile frontend assets before `go build` when UI is embedded
