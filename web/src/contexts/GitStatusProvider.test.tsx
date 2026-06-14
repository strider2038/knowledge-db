/**
 * @vitest-environment jsdom
 */
import { describe, expect, it, vi } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import { GitStatusProvider } from './GitStatusProvider'
import { useGitStatus } from '@/hooks/useGitStatus'

vi.mock('@/services/api', () => ({
  getGitStatus: vi.fn().mockResolvedValue({ has_changes: false, changed_files: 0, git_disabled: false }),
}))

function RevisionProbe() {
  const { dataRevision, bumpDataRevision } = useGitStatus()
  return (
    <div>
      <span data-testid="revision">{dataRevision}</span>
      <button type="button" onClick={bumpDataRevision}>
        bump
      </button>
    </div>
  )
}

describe('GitStatusProvider', () => {
  it('exposes dataRevision and bumpDataRevision', async () => {
    render(
      <GitStatusProvider>
        <RevisionProbe />
      </GitStatusProvider>,
    )

    expect(screen.getByTestId('revision')).toHaveTextContent('0')

    await act(async () => {
      screen.getByRole('button', { name: 'bump' }).click()
    })

    expect(screen.getByTestId('revision')).toHaveTextContent('1')
  })
})
