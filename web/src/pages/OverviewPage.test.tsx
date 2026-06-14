/**
 * @vitest-environment jsdom
 */
import { describe, expect, it, vi, beforeEach, type MockedFunction } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { TooltipProvider } from '@/components/ui/tooltip'
import { GitStatusProvider } from '@/hooks/useGitStatus'
import { OverviewPage } from './OverviewPage'
import type { GetNodesParams, NodeListItem } from '../services/api'

type GetNodesWithParamsFn = (params: GetNodesParams) => Promise<{
  nodes: NodeListItem[]
  total: number
}>

vi.mock('../services/api', () => ({
  getTree: vi.fn().mockResolvedValue({
    name: '',
    path: '',
    children: [{ name: 'topic', path: 'topic', children: [] }],
  }),
  getLabelSuggestions: vi.fn().mockResolvedValue(['favorite', 'review']),
  getNodesWithParams: vi.fn().mockResolvedValue({
    nodes: [
      {
        path: 'topic/node1',
        title: 'Node 1',
        type: 'article',
        created: '2024-01-01T00:00:00Z',
        source_url: 'https://example.com',
        labels: ['favorite'],
        manual_processed: false,
      },
    ],
    total: 1,
  }),
  getGitStatus: vi.fn().mockResolvedValue({ has_changes: false, changed_files: 0, git_disabled: false }),
}))

function renderOverview(initialEntry = '/') {
  return render(
    <GitStatusProvider>
      <TooltipProvider>
        <MemoryRouter initialEntries={[initialEntry]}>
          <Routes>
            <Route path="/" element={<OverviewPage />} />
          </Routes>
        </MemoryRouter>
      </TooltipProvider>
    </GitStatusProvider>
  )
}

/** Последний вызов списка узлов (limit 50), не запрос дерева (limit 10000). */
function lastMainListCall(
  getNodesWithParams: MockedFunction<GetNodesWithParamsFn>
): GetNodesParams | undefined {
  const main = getNodesWithParams.mock.calls
    .map((c) => c[0])
    .filter((p) => p.limit === 50)
  return main[main.length - 1]
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

  it('renders node link when at overview with query params', async () => {
    renderOverview('/?path=topic&type=article')
    const links = await screen.findAllByRole('link', { name: 'Node 1' })
    expect(links[0]).toHaveAttribute('href', '/node/topic/node1')
  })

  it('when URL has manual_processed=false, passes false to getNodesWithParams for main list', async () => {
    const { getNodesWithParams } = await import('../services/api')
    const getNodes = vi.mocked(getNodesWithParams)
    renderOverview('/?manual_processed=false')
    await screen.findByText('Node 1')
    await waitFor(() => {
      const p = lastMainListCall(getNodes)
      expect(p?.manual_processed).toBe(false)
    })
  })

  it('when URL has manual_processed=true, passes true to getNodesWithParams for main list', async () => {
    const { getNodesWithParams } = await import('../services/api')
    const getNodes = vi.mocked(getNodesWithParams)
    renderOverview('/?manual_processed=true')
    await screen.findByText('Node 1')
    await waitFor(() => {
      const p = lastMainListCall(getNodes)
      expect(p?.manual_processed).toBe(true)
    })
  })

  it('when URL has invalid manual_processed, does not filter (undefined)', async () => {
    const { getNodesWithParams } = await import('../services/api')
    const getNodes = vi.mocked(getNodesWithParams)
    renderOverview('/?manual_processed=yes')
    await screen.findByText('Node 1')
    await waitFor(() => {
      const p = lastMainListCall(getNodes)
      expect(p?.manual_processed).toBeUndefined()
    })
  })

  it('when user selects «Проверено вручную», requests manual_processed true', async () => {
    const { getNodesWithParams } = await import('../services/api')
    const getNodes = vi.mocked(getNodesWithParams)
    renderOverview()
    await screen.findByText('Node 1')
    getNodes.mockClear()

    const trigger = screen.getByLabelText('Фильтр по ручной проверке')
    fireEvent.pointerDown(trigger)
    fireEvent.click(trigger)
    await waitFor(() => {
      fireEvent.click(screen.getByRole('menuitemradio', { name: 'Проверено вручную' }))
    })

    await waitFor(() => {
      const p = lastMainListCall(getNodes)
      expect(p?.manual_processed).toBe(true)
    })
  })

  it('when URL has labels filter, passes labels to getNodesWithParams', async () => {
    const { getNodesWithParams } = await import('../services/api')
    const getNodes = vi.mocked(getNodesWithParams)
    renderOverview('/?labels=favorite,review')
    await screen.findByText('Node 1')
    await waitFor(() => {
      const p = lastMainListCall(getNodes)
      expect(p?.labels).toEqual(['favorite', 'review'])
    })
  })

  it('renders label chips in table', async () => {
    renderOverview()
    await screen.findByText('Node 1')
    const chips = await screen.findAllByText('favorite')
    expect(chips.length).toBeGreaterThanOrEqual(1)
  })

  it('when user selects «Все» after filtered, omits manual_processed from main list params', async () => {
    const { getNodesWithParams } = await import('../services/api')
    const getNodes = vi.mocked(getNodesWithParams)
    renderOverview('/?manual_processed=false')
    await screen.findByText('Node 1')

    getNodes.mockClear()
    const trigger = screen.getByLabelText('Фильтр по ручной проверке')
    fireEvent.pointerDown(trigger)
    fireEvent.click(trigger)
    await waitFor(() => {
      fireEvent.click(screen.getByRole('menuitemradio', { name: 'Все записи' }))
    })

    await waitFor(() => {
      const p = lastMainListCall(getNodes)
      expect(p?.manual_processed).toBeUndefined()
    })
  })
})
