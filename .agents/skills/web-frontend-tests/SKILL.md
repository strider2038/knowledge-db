---
name: web-frontend-tests
description: Testing React code in web/ with Vitest and React Testing Library. Use when writing or updating component, page, hook, or service tests.
---

# Frontend tests — web/

Framework: **Vitest** + **@testing-library/react** + **@testing-library/jest-dom**.

## Commands

```bash
cd web && npm test              # single run
cd web && npm run test -- --watch   # watch (if script added)
# repo root:
task web:test
```

Vitest config is in `web/vite.config.ts` (or vitest section).

## File layout

Tests live **next to** the module under test:

```text
web/src/pages/AddPage.tsx
web/src/pages/AddPage.test.tsx
web/src/services/api.ts
web/src/services/api.test.ts
```

## Environment

Page tests that need DOM:

```tsx
/**
 * @vitest-environment jsdom
 */
```

## Router wrapper

Pages need `MemoryRouter` (and routes) in tests:

```tsx
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { AddPage } from './AddPage'

function renderAddPage() {
  return render(
    <MemoryRouter initialEntries={['/add']}>
      <Routes>
        <Route path="/add" element={<AddPage />} />
      </Routes>
    </MemoryRouter>,
  )
}
```

## Mocking API

Mock `../services/api` (or the module your page imports):

```tsx
const { ingestText } = vi.hoisted(() => ({
  ingestText: vi.fn(),
}))

vi.mock('../services/api', () => ({
  ingestText,
}))

beforeEach(() => {
  vi.clearAllMocks()
})
```

## Assertions

- Prefer `screen.getByRole`, `getByLabelText`, `getByPlaceholderText` over test ids
- `userEvent` for interactions when needed (`@testing-library/user-event`)
- `expect(...).toBeInTheDocument()` from jest-dom

Example from `AddPage.test.tsx`:

```tsx
renderAddPage()
expect(screen.getByRole('button', { name: 'Добавить' })).toBeInTheDocument()
fireEvent.click(screen.getByRole('button', { name: 'Добавить' }))
expect(ingestText).toHaveBeenCalled()
```

(UI copy may be Russian — match actual button labels.)

## Hook / pure function tests

No router required:

```tsx
import { describe, expect, it } from 'vitest'
import { parseHeadings } from './headings'
```

## What we do not use

- Miniapp `renderWithTheme` / styled-components theme wrapper — use plain `render` or wrap only what the component needs (e.g. `ThemeProvider` from `next-themes` if required)

## Checklist

- [ ] `@vitest-environment jsdom` for components using DOM APIs
- [ ] API modules mocked at import path used by the component
- [ ] Router provided for routed pages
- [ ] Tests run via `npm test` in `web/`
