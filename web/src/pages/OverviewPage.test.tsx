/**
 * @vitest-environment jsdom
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { TooltipProvider } from '@/components/ui/tooltip'
import { OverviewPage } from './OverviewPage'

vi.mock('../services/api', () => ({
  getTree: vi.fn().mockResolvedValue({
    name: '',
    path: '',
    children: [{ name: 'topic', path: 'topic', children: [] }],
  }),
  getNodesWithParams: vi.fn().mockResolvedValue({
    nodes: [
      {
        path: 'topic/node1',
        title: 'Node 1',
        type: 'article',
        created: '2024-01-01T00:00:00Z',
        source_url: 'https://example.com',
      },
    ],
    total: 1,
  }),
}))

function renderOverview() {
  return render(
    <TooltipProvider>
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="/" element={<OverviewPage />} />
        </Routes>
      </MemoryRouter>
    </TooltipProvider>
  )
}

describe('OverviewPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders tree and table after loading', async () => {
    renderOverview()
    expect(await screen.findByText('Вся база')).toBeTruthy()
    expect(await screen.findByText('Node 1')).toBeTruthy()
    expect(await screen.findByRole('button', { name: 'topic' })).toBeTruthy()
  })

  it('calls getNodesWithParams with recursive=true', async () => {
    const { getNodesWithParams } = await import('../services/api')
    renderOverview()
    await screen.findByText('Node 1')
    expect(getNodesWithParams).toHaveBeenCalledWith(
      expect.objectContaining({ recursive: true })
    )
  })
})
