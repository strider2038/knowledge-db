import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { NodeAgentEditDialog } from './NodeAgentEditDialog'
import * as api from '@/services/api'

vi.mock('@/services/api', () => ({
  startNodeAgentEdit: vi.fn(),
}))

describe('NodeAgentEditDialog', () => {
  beforeEach(() => {
    vi.mocked(api.startNodeAgentEdit).mockReset()
    sessionStorage.clear()
  })

  it('disables submit when instruction is empty', () => {
    render(
      <NodeAgentEditDialog open nodePath="topic/node" onOpenChange={() => {}} onStarted={() => {}} />,
    )
    expect(screen.getByRole('button', { name: 'Запустить' })).toBeDisabled()
  })

  it('starts operation and closes on success', async () => {
    const onStarted = vi.fn()
    const onOpenChange = vi.fn()
    vi.mocked(api.startNodeAgentEdit).mockResolvedValue({
      id: 'op-1',
      node_path: 'topic/node',
      status: 'running',
      stage: 'edit',
      started_at: new Date().toISOString(),
      sync_done: false,
      edit_ok: false,
    })

    render(
      <NodeAgentEditDialog
        open
        nodePath="topic/node"
        onOpenChange={onOpenChange}
        onStarted={onStarted}
      />,
    )

    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'add summary' } })
    fireEvent.click(screen.getByRole('button', { name: 'Запустить' }))

    await waitFor(() => {
      expect(api.startNodeAgentEdit).toHaveBeenCalledWith('topic/node', 'add summary')
      expect(onStarted).toHaveBeenCalledWith('op-1')
      expect(onOpenChange).toHaveBeenCalledWith(false)
    })
    expect(sessionStorage.getItem('kb:agent-edit-instruction:topic/node')).toBe('add summary')
  })

  it('restores saved instruction when reopened', () => {
    sessionStorage.setItem('kb:agent-edit-instruction:topic/node', 'previous task')
    render(
      <NodeAgentEditDialog open nodePath="topic/node" onOpenChange={() => {}} onStarted={() => {}} />,
    )
    expect(screen.getByRole('textbox')).toHaveValue('previous task')
  })

  it('shows error when start fails', async () => {
    vi.mocked(api.startNodeAgentEdit).mockRejectedValue(new Error('agent unavailable'))

    render(
      <NodeAgentEditDialog open nodePath="topic/node" onOpenChange={() => {}} onStarted={() => {}} />,
    )

    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'fix typo' } })
    fireEvent.click(screen.getByRole('button', { name: 'Запустить' }))

    expect(await screen.findByText('agent unavailable')).toBeInTheDocument()
  })
})
