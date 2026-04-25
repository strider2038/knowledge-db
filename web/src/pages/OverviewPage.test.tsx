/**
 * @vitest-environment jsdom
 */
import { describe, expect, it, vi, beforeEach, type MockedFunction } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { TooltipProvider } from '@/components/ui/tooltip'
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
  getNodesWithParams: vi.fn().mockResolvedValue({
    nodes: [
      {
        path: 'topic/node1',
        title: 'Node 1',
        type: 'article',
        created: '2024-01-01T00:00:00Z',
        source_url: 'https://example.com',
        manual_processed: false,
      },
    ],
    total: 1,
  }),
}))

function renderOverview(initialEntry = '/') {
  return render(
    <TooltipProvider>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/" element={<OverviewPage />} />
        </Routes>
      </MemoryRouter>
    </TooltipProvider>
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
    render(
      <TooltipProvider>
        <MemoryRouter initialEntries={['/?path=topic&type=article']}>
          <Routes>
            <Route path="/" element={<OverviewPage />} />
          </Routes>
        </MemoryRouter>
      </TooltipProvider>
    )
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

    const select = screen.getByLabelText('Фильтр по ручной проверке')
    fireEvent.change(select, { target: { value: 'true' } })

    await waitFor(() => {
      const p = lastMainListCall(getNodes)
      expect(p?.manual_processed).toBe(true)
    })
  })

  it('when user selects «Все» after filtered, omits manual_processed from main list params', async () => {
    const { getNodesWithParams } = await import('../services/api')
    const getNodes = vi.mocked(getNodesWithParams)
    renderOverview('/?manual_processed=false')
    await screen.findByText('Node 1')

    getNodes.mockClear()
    const select = screen.getByLabelText('Фильтр по ручной проверке')
    fireEvent.change(select, { target: { value: '' } })

    await waitFor(() => {
      const p = lastMainListCall(getNodes)
      expect(p?.manual_processed).toBeUndefined()
    })
  })
})
