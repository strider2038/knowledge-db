---
name: web-frontend-tests
description: Testing React with Vitest and React Testing Library — component, page, hook, and service tests.
---

# Frontend tests (Vitest + React Testing Library)

Framework: **Vitest** + **@testing-library/react** + **@testing-library/jest-dom**.

## Commands

```bash
cd web && npm test              # single run
cd web && npm run test -- --watch   # watch (if script added)
```

Vitest config is in `vite.config.ts` (or a dedicated vitest section).

## File layout

Tests live **next to** the module under test:

```text
src/pages/EditPage.tsx
src/pages/EditPage.test.tsx
src/services/api.ts
src/services/api.test.ts
```

## Environment

Page tests that need DOM:

```tsx
/**
 * @vitest-environment jsdom
 */
```

## Router wrapper

Routed pages need `MemoryRouter` (and routes) in tests:

```tsx
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { EditPage } from './EditPage'

function renderEditPage() {
  return render(
    <MemoryRouter initialEntries={['/edit']}>
      <Routes>
        <Route path="/edit" element={<EditPage />} />
      </Routes>
    </MemoryRouter>,
  )
}
```

## Mocking API

Mock the API module your page imports:

```tsx
const { updateResource } = vi.hoisted(() => ({
  updateResource: vi.fn(),
}))

vi.mock('../services/api', () => ({
  updateResource,
}))

beforeEach(() => {
  vi.clearAllMocks()
})
```

## Assertions

- Prefer `screen.getByRole`, `getByLabelText`, `getByPlaceholderText` over test ids
- `userEvent` for interactions when needed (`@testing-library/user-event`)
- `expect(...).toBeInTheDocument()` from jest-dom

```tsx
renderEditPage()
expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument()
fireEvent.click(screen.getByRole('button', { name: 'Save' }))
expect(updateResource).toHaveBeenCalled()
```

Match actual button labels and copy from the component under test.

## Hook / pure function tests

No router required:

```tsx
import { describe, expect, it } from 'vitest'
import { parseHeadings } from './headings'
```

## What we do not use

- Framework-specific theme wrappers that the app does not use — wrap only what the component needs (e.g. a `ThemeProvider` when the component depends on it)

## Checklist

- [ ] `@vitest-environment jsdom` for components using DOM APIs
- [ ] API modules mocked at import path used by the component
- [ ] Router provided for routed pages
- [ ] Tests run via `npm test` in the frontend package
