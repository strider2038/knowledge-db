/**
 * @vitest-environment jsdom
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { TooltipProvider } from '@/components/ui/tooltip'
import { GitStatusProvider } from '@/hooks/useGitStatus'
import { NodePage } from './NodePage'

const { mockNode, mockNavigate, getNode, patchNodeManualProcessed, startJob, getJobStatus, startNodeNormalization, getNodeNormalizationStatus, getNodeNormalizationLogs, startNodeDumpImages, getNodeDumpImagesStatus, getNodeDumpImagesLogs } = vi.hoisted(() => {
  const mockNode = {
    path: 'programming/scaling/load-balancing',
    annotation: 'Annotation **text**',
    content: 'Content with `code`',
    metadata: {
      title: 'Load Balancing',
      type: 'article',
      created: '2024-01-15T00:00:00Z',
      updated: '2024-03-01T00:00:00Z',
      source_url: 'https://example.com/article',
      source_author: 'Author Name',
      source_date: '2024-01-10',
      keywords: ['load-balancing', 'scaling'],
      manual_processed: false,
    },
  }
  return {
    mockNode,
    mockNavigate: vi.fn(),
    getNode: vi.fn().mockResolvedValue(mockNode),
    patchNodeManualProcessed: vi.fn().mockImplementation(async (_path: string, v: boolean) => ({
      ...mockNode,
      metadata: { ...mockNode.metadata, manual_processed: v },
    })),
    startJob: vi.fn().mockResolvedValue({
      id: 'job-refresh-1',
      type: 'refresh_description',
      target: mockNode.path,
      status: 'running',
      stage: 'start',
      started_at: new Date().toISOString(),
      next_offset: 0,
    }),
    getJobStatus: vi.fn().mockResolvedValue({
      id: 'job-refresh-1',
      type: 'refresh_description',
      target: mockNode.path,
      status: 'success',
      stage: 'done',
      started_at: new Date().toISOString(),
      finished_at: new Date().toISOString(),
      next_offset: 0,
    }),
    startNodeNormalization: vi.fn().mockResolvedValue({
      id: 'op-1',
      node_path: mockNode.path,
      status: 'running',
      stage: 'normalize',
      started_at: new Date().toISOString(),
      sync_done: false,
      normalize_ok: false,
    }),
    getNodeNormalizationStatus: vi.fn().mockResolvedValue({
      id: 'op-1',
      node_path: mockNode.path,
      status: 'success',
      stage: 'done',
      started_at: new Date().toISOString(),
      finished_at: new Date().toISOString(),
      sync_done: true,
      normalize_ok: true,
    }),
    getNodeNormalizationLogs: vi.fn().mockResolvedValue({ entries: [], next_offset: 0 }),
    startNodeDumpImages: vi.fn().mockResolvedValue({
      id: 'dump-1',
      node_path: mockNode.path,
      status: 'running',
      stage: 'dump',
      started_at: new Date().toISOString(),
      sync_done: false,
      dump_ok: false,
    }),
    getNodeDumpImagesStatus: vi.fn().mockResolvedValue({
      id: 'dump-1',
      node_path: mockNode.path,
      status: 'success',
      stage: 'done',
      started_at: new Date().toISOString(),
      finished_at: new Date().toISOString(),
      sync_done: true,
      dump_ok: true,
    }),
    getNodeDumpImagesLogs: vi.fn().mockResolvedValue({ entries: [], next_offset: 0 }),
  }
})

vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>()
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

vi.mock('../services/api', () => ({
  getNode,
  patchNodeManualProcessed,
  startJob,
  getJobStatus,
  startNodeNormalization,
  getNodeNormalizationStatus,
  getNodeNormalizationLogs,
  startNodeDumpImages,
  getNodeDumpImagesStatus,
  getNodeDumpImagesLogs,
  getGitStatus: vi.fn().mockResolvedValue({ has_changes: false, changed_files: 0 }),
}))

function renderNodePage(initialPath = '/node/programming/scaling/load-balancing', state?: { returnTo: string }) {
  const result = render(
    <GitStatusProvider>
      <TooltipProvider>
        <MemoryRouter
          initialEntries={[{ pathname: initialPath, state }]}
          initialIndex={0}
        >
          <Routes>
            <Route path="/node/*" element={<NodePage />} />
          </Routes>
        </MemoryRouter>
      </TooltipProvider>
    </GitStatusProvider>
  )
  return result
}

describe('NodePage', () => {
  beforeEach(() => {
    vi.useRealTimers()
    mockNavigate.mockClear()
    getNode.mockReset()
    getNode.mockResolvedValue(mockNode)
    patchNodeManualProcessed.mockReset()
    patchNodeManualProcessed.mockImplementation(async (_path: string, v: boolean) => ({
      ...mockNode,
      metadata: { ...mockNode.metadata, manual_processed: v },
    }))
    startJob.mockReset()
    startJob.mockResolvedValue({
      id: 'job-refresh-1',
      type: 'refresh_description',
      target: mockNode.path,
      status: 'running',
      stage: 'start',
      started_at: new Date().toISOString(),
      next_offset: 0,
    })
    getJobStatus.mockReset()
    getJobStatus.mockResolvedValue({
      id: 'job-refresh-1',
      type: 'refresh_description',
      target: mockNode.path,
      status: 'success',
      stage: 'done',
      started_at: new Date().toISOString(),
      finished_at: new Date().toISOString(),
      next_offset: 0,
    })
    startNodeNormalization.mockReset()
    startNodeNormalization.mockResolvedValue({
      id: 'op-1',
      node_path: mockNode.path,
      status: 'running',
      stage: 'normalize',
      started_at: new Date().toISOString(),
      sync_done: false,
      normalize_ok: false,
    })
    getNodeNormalizationStatus.mockReset()
    getNodeNormalizationStatus.mockResolvedValue({
      id: 'op-1',
      node_path: mockNode.path,
      status: 'success',
      stage: 'done',
      started_at: new Date().toISOString(),
      finished_at: new Date().toISOString(),
      sync_done: true,
      normalize_ok: true,
    })
    getNodeNormalizationLogs.mockReset()
    getNodeNormalizationLogs.mockResolvedValue({ entries: [], next_offset: 0 })
    startNodeDumpImages.mockReset()
    startNodeDumpImages.mockResolvedValue({
      id: 'dump-1',
      node_path: mockNode.path,
      status: 'running',
      stage: 'dump',
      started_at: new Date().toISOString(),
      sync_done: false,
      dump_ok: false,
    })
    getNodeDumpImagesStatus.mockReset()
    getNodeDumpImagesStatus.mockResolvedValue({
      id: 'dump-1',
      node_path: mockNode.path,
      status: 'success',
      stage: 'done',
      started_at: new Date().toISOString(),
      finished_at: new Date().toISOString(),
      sync_done: true,
      dump_ok: true,
    })
    getNodeDumpImagesLogs.mockReset()
    getNodeDumpImagesLogs.mockResolvedValue({ entries: [], next_offset: 0 })
  })

  it('marks manual processed via Проверено button', async () => {
    renderNodePage()
    expect(await screen.findByRole('heading', { level: 1, name: 'Load Balancing' })).toBeInTheDocument()
    const btn = screen.getByRole('button', { name: 'Проверено' })
    expect(btn).toHaveAttribute('data-variant', 'outline')
    fireEvent.click(btn)
    await waitFor(() => {
      expect(patchNodeManualProcessed).toHaveBeenCalledWith('programming/scaling/load-balancing', true)
    })
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Проверено' })).toHaveAttribute('data-variant', 'ghost')
    })
  })

  it('shows check when already manual processed', async () => {
    getNode.mockResolvedValue({
      ...mockNode,
      metadata: { ...mockNode.metadata, manual_processed: true },
    })
    renderNodePage()
    const btn = await screen.findByRole('button', { name: 'Проверено' })
    expect(btn).toHaveAttribute('data-variant', 'ghost')
  })

  it('renders title, type badge, breadcrumbs, annotation, content, keywords; no Metadata block', async () => {
    renderNodePage()
    expect(await screen.findByRole('heading', { level: 1, name: 'Load Balancing' })).toBeInTheDocument()
    expect(screen.getByText('article')).toBeInTheDocument()
    expect(screen.getByText('Обзор')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'programming' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'scaling' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'load-balancing' })).toBeInTheDocument()
    expect(screen.getAllByText(/Annotation/).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/Content with/).length).toBeGreaterThan(0)
    expect(screen.queryByText('Метаданные')).not.toBeInTheDocument()
    expect(screen.getAllByText('load-balancing').length).toBeGreaterThan(0)
    expect(screen.getAllByText('scaling').length).toBeGreaterThan(0)
    expect(screen.getByRole('button', { name: 'Сообщить о проблеме' })).toBeInTheDocument()
  })

  it('back button navigates to / when state is absent', async () => {
    renderNodePage('/node/programming/scaling/load-balancing')
    const backBtns = await screen.findAllByText('← Назад')
    fireEvent.click(backBtns[0])
    expect(mockNavigate).toHaveBeenCalled()
    expect(mockNavigate).toHaveBeenCalledWith('/')
  })

  it('back button navigates to returnTo when coming from overview', async () => {
    const returnTo = '/?path=programming&type=article&page=1'
    renderNodePage('/node/programming/scaling/load-balancing', { returnTo })
    const backBtns = await screen.findAllByText('← Назад')
    fireEvent.click(backBtns[0])
    expect(mockNavigate).toHaveBeenCalledWith(returnTo)
  })

  it('shows source attribution line (url, author, date) for note with source fields', async () => {
    getNode.mockResolvedValue({
      ...mockNode,
      path: 'microservices/messaging/gde-mozhet-poteratsya-exactly-once',
      metadata: {
        ...mockNode.metadata,
        type: 'note',
        title: 'Где может потеряться "exactly-once"',
        source_url: 'https://example.com/post',
        source_author: 'Иван Петров',
        source_date: '2026-03-01',
      },
      annotation: 'Заметка о exactly-once.',
      content: 'Контент заметки.',
    })
    renderNodePage('/node/microservices/messaging/gde-mozhet-poteratsya-exactly-once')
    expect(await screen.findByRole('heading', { level: 1, name: 'Где может потеряться "exactly-once"' })).toBeInTheDocument()
    const sourceLink = screen.getByRole('link', { name: /https:\/\/example\.com\/post/ })
    expect(sourceLink).toHaveAttribute('href', 'https://example.com/post')
    expect(screen.getByText(/Автор: Иван Петров/)).toBeInTheDocument()
    expect(screen.getByText(/Дата источника:/)).toBeInTheDocument()
  })

  it('for type link: shows clickable source link before annotation, hides content block', async () => {
    getNode.mockResolvedValue({
      ...mockNode,
      metadata: {
        ...mockNode.metadata,
        type: 'link',
        title: 'Component Gallery',
        source_url: 'https://example.com/gallery',
      },
      content: '',
      annotation: 'UI components collection.',
    })
    renderNodePage('/node/ui/component-gallery-ui')
    expect(await screen.findByRole('heading', { level: 1, name: 'Component Gallery' })).toBeInTheDocument()
    const links = screen
      .getAllByRole('link')
      .filter((l) => l.getAttribute('href') === 'https://example.com/gallery')
    const sourceLink = links.find((l) => l.classList.contains('rounded-lg')) ?? links[0]
    expect(sourceLink).toHaveAttribute('href', 'https://example.com/gallery')
    expect(sourceLink).toHaveAttribute('target', '_blank')
    expect(screen.getByText(/UI components collection/)).toBeInTheDocument()
    expect(screen.queryByText('Содержание')).not.toBeInTheDocument()
  })

  it('refreshes description from source and updates current node', async () => {
    const setIntervalSpy = vi.spyOn(globalThis, 'setInterval').mockImplementation((cb: TimerHandler) => {
      if (typeof cb === 'function') cb()
      return 1 as unknown as ReturnType<typeof setInterval>
    })
    const clearIntervalSpy = vi.spyOn(globalThis, 'clearInterval').mockImplementation(() => {})
    getNode
      .mockResolvedValueOnce(mockNode)
      .mockResolvedValueOnce({
        ...mockNode,
        annotation: 'Updated annotation',
        content: 'Updated content',
        metadata: { ...mockNode.metadata, title: 'Updated title', type: 'note' },
      })

    renderNodePage()

    const btn = await screen.findByRole('button', { name: 'Обновить описание из источника' })
    fireEvent.click(btn)

    await waitFor(() => {
      expect(startJob).toHaveBeenCalledWith('refresh_description', 'programming/scaling/load-balancing')
    })
    await waitFor(() => {
      expect(getJobStatus).toHaveBeenCalledWith('job-refresh-1')
    })
    expect(await screen.findByRole('heading', { level: 1, name: 'Updated title' })).toBeInTheDocument()
    expect(screen.getByText(/Updated annotation/)).toBeInTheDocument()
    expect(screen.getByText('Описание обновлено')).toBeInTheDocument()
    setIntervalSpy.mockRestore()
    clearIntervalSpy.mockRestore()
  })

  it('hides refresh description action when source_url is absent', async () => {
    getNode.mockResolvedValue({
      ...mockNode,
      metadata: {
        ...mockNode.metadata,
        source_url: undefined,
      },
    })

    renderNodePage()

    expect(await screen.findByRole('heading', { level: 1, name: 'Load Balancing' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Обновить описание из источника' })).not.toBeInTheDocument()
  })

  it('runs node normalization and shows success', async () => {
    getNodeNormalizationLogs.mockResolvedValue({
      entries: [{ offset: 1, stream: 'stdout', text: 'line one', timestamp: new Date().toISOString() }],
      next_offset: 1,
    })
    renderNodePage()

    const btn = await screen.findByRole('button', { name: 'Нормализация' })
    fireEvent.click(btn)

    await waitFor(() => {
      expect(startNodeNormalization).toHaveBeenCalledWith('programming/scaling/load-balancing')
    })
    await waitFor(() => {
      expect(getNodeNormalizationStatus).toHaveBeenCalledWith('op-1')
    })
    expect(await screen.findByText(/Логи нормализации ·/)).toBeInTheDocument()
    expect(await screen.findByText('line one')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /Логи нормализации ·/ }))
    expect(screen.getByText('Режим логов')).toBeInTheDocument()
    expect(await screen.findByText('Нормализация завершена')).toBeInTheDocument()
  })

  it('shows error status in normalization log panel', async () => {
    getNodeNormalizationStatus.mockResolvedValue({
      id: 'op-1',
      node_path: mockNode.path,
      status: 'error',
      stage: 'normalize',
      error: 'normalize failed',
      started_at: new Date().toISOString(),
      finished_at: new Date().toISOString(),
      sync_done: false,
      normalize_ok: false,
    })

    renderNodePage()
    const btn = await screen.findByRole('button', { name: 'Нормализация' })
    fireEvent.click(btn)

    expect(await screen.findByText('normalize failed')).toBeInTheDocument()
    expect(await screen.findByText('Логи нормализации · error')).toBeInTheDocument()
  })

  it('runs dump images and shows success', async () => {
    getNodeDumpImagesLogs.mockResolvedValue({
      entries: [{ offset: 1, stream: 'stdout', text: 'downloaded image', timestamp: new Date().toISOString() }],
      next_offset: 1,
    })
    renderNodePage()

    const btn = await screen.findByRole('button', { name: 'Выгрузить изображения' })
    fireEvent.click(btn)

    await waitFor(() => {
      expect(startNodeDumpImages).toHaveBeenCalledWith('programming/scaling/load-balancing')
    })
    await waitFor(() => {
      expect(getNodeDumpImagesStatus).toHaveBeenCalledWith('dump-1')
    })
    expect(await screen.findByText(/Логи выгрузки изображений ·/)).toBeInTheDocument()
    expect(await screen.findByText('downloaded image')).toBeInTheDocument()
    expect(await screen.findByText('Выгрузка изображений завершена')).toBeInTheDocument()
  })

  it('shows error status in dump images log panel', async () => {
    getNodeDumpImagesStatus.mockResolvedValue({
      id: 'dump-1',
      node_path: mockNode.path,
      status: 'error',
      stage: 'dump',
      error: 'dump failed',
      started_at: new Date().toISOString(),
      finished_at: new Date().toISOString(),
      sync_done: false,
      dump_ok: false,
    })

    renderNodePage()
    const btn = await screen.findByRole('button', { name: 'Выгрузить изображения' })
    fireEvent.click(btn)

    expect(await screen.findByText('dump failed')).toBeInTheDocument()
    expect(await screen.findByText('Логи выгрузки изображений · error')).toBeInTheDocument()
  })
})
