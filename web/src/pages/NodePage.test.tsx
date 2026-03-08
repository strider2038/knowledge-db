/**
 * @vitest-environment jsdom
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { TooltipProvider } from '@/components/ui/tooltip'
import { NodePage } from './NodePage'

const { mockNode, mockNavigate, getNode } = vi.hoisted(() => {
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
    },
  }
  return {
    mockNode,
    mockNavigate: vi.fn(),
    getNode: vi.fn().mockResolvedValue(mockNode),
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
}))

function renderNodePage(initialPath = '/node/programming/scaling/load-balancing', state?: { returnTo: string }) {
  const result = render(
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
  )
  return result
}

describe('NodePage', () => {
  beforeEach(() => {
    mockNavigate.mockClear()
    getNode.mockResolvedValue(mockNode)
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
})
