/**
 * @vitest-environment jsdom
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { SearchPage } from './SearchPage'

type SearchKnowledgeBase = typeof import('@/services/api')['searchKnowledgeBase']

const { searchKnowledgeBase } = vi.hoisted(() => ({
  searchKnowledgeBase: vi.fn<SearchKnowledgeBase>(),
}))

vi.mock('@/services/api', () => ({
  searchKnowledgeBase,
}))

function renderSearchPage(initialEntry = '/search?q=sqlite') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/search" element={<SearchPage />} />
        <Route path="/chat" element={<div>Chat route</div>} />
        <Route path="/node/*" element={<div>Node route</div>} />
      </Routes>
    </MemoryRouter>
  )
}

describe('SearchPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders search results with fragments', async () => {
    searchKnowledgeBase.mockResolvedValue({
      results: [
        {
          path: 'articles/sqlite',
          title: 'SQLite',
          type: 'article',
          annotation: 'Local database',
          keywords: ['sqlite'],
          score: 1,
          rank: 1,
          match_reasons: ['keywords'],
          source_kinds: ['keyword'],
          fragments: [{ heading: 'Intro', snippet: 'sqlite snippet', score: 1, match_type: 'keyword' }],
        },
      ],
      total: 1,
      query: 'sqlite',
      mode: 'search',
      meta: { keyword_index: 'fts5', query_rewrite: 'sqlite local index' },
    })

    renderSearchPage()

    expect(await screen.findByText('SQLite')).toBeInTheDocument()
    expect(screen.getByText('Local database')).toBeInTheDocument()
    expect(screen.getByText('sqlite snippet')).toBeInTheDocument()
    expect(screen.getByText('Как выполнен поиск')).toBeInTheDocument()
    expect(screen.getByText('Исходный запрос')).toBeInTheDocument()
    expect(screen.getByText('Запрос к индексу')).toBeInTheDocument()
    expect(screen.getByText('sqlite local index')).toBeInTheDocument()
    expect(screen.getByText('score 1.000')).toBeInTheDocument()
    expect(screen.getByText('reason: keywords')).toBeInTheDocument()
    expect(screen.getByText('source: keyword')).toBeInTheDocument()
    expect(screen.getAllByText('keyword').length).toBeGreaterThan(0)
    expect(screen.getAllByText('1.000').length).toBeGreaterThan(0)
  })

  it('shows empty state', async () => {
    searchKnowledgeBase.mockResolvedValue({
      results: [],
      total: 0,
      query: 'missing',
      mode: 'search',
      meta: { keyword_index: 'scan' },
    })

    renderSearchPage('/search?q=missing')

    expect(await screen.findByText('Ничего не найдено.')).toBeInTheDocument()
  })

  it('collapses results after a strong score drop', async () => {
    searchKnowledgeBase.mockResolvedValue({
      results: [
        {
          path: 'articles/strong',
          title: 'Strong match',
          type: 'article',
          annotation: '',
          keywords: [],
          score: 1,
          rank: 1,
          match_reasons: ['exact_token'],
          source_kinds: ['exact'],
          fragments: [],
        },
        {
          path: 'articles/close',
          title: 'Close match',
          type: 'article',
          annotation: '',
          keywords: [],
          score: 0.9,
          rank: 2,
          match_reasons: ['vector'],
          source_kinds: ['vector_node'],
          fragments: [],
        },
        {
          path: 'articles/tail',
          title: 'Tail match',
          type: 'article',
          annotation: 'Hidden until expanded',
          keywords: [],
          score: 0.4,
          rank: 3,
          match_reasons: ['vector'],
          source_kinds: ['vector_node'],
          fragments: [],
        },
      ],
      total: 3,
      query: 'sqlite',
      mode: 'search',
      meta: { keyword_index: 'fts5' },
    })

    const { container } = renderSearchPage()

    expect(await screen.findByText('Strong match')).toBeInTheDocument()
    expect(screen.getByText('Ниже заметный перепад score: результаты показаны свернуто.')).toBeInTheDocument()
    expect(container.querySelectorAll('details').length).toBeGreaterThanOrEqual(2)
  })

  it('shows unavailable error', async () => {
    searchKnowledgeBase.mockRejectedValue(new Error('embedding service unavailable'))

    renderSearchPage()

    expect(await screen.findByText('embedding service unavailable')).toBeInTheDocument()
  })

  it('navigates to chat with result sources', async () => {
    searchKnowledgeBase.mockResolvedValue({
      results: [
        {
          path: 'articles/sqlite',
          title: 'SQLite',
          type: 'article',
          annotation: '',
          keywords: [],
          score: 1,
          rank: 1,
          match_reasons: [],
          source_kinds: [],
          fragments: [],
        },
      ],
      total: 1,
      query: 'sqlite',
      mode: 'search',
      meta: { keyword_index: 'fts5' },
    })

    renderSearchPage()
    await screen.findByText('SQLite')
    fireEvent.click(screen.getByRole('button', { name: /Спросить по этим источникам/ }))

    await waitFor(() => expect(screen.getByText('Chat route')).toBeInTheDocument())
  })

  it('keeps query and sends type filter when type button is selected', async () => {
    searchKnowledgeBase.mockResolvedValue({
      results: [],
      total: 0,
      query: 'sqlite',
      mode: 'search',
      meta: { keyword_index: 'fts5' },
    })

    renderSearchPage()
    await waitFor(() => expect(searchKnowledgeBase).toHaveBeenCalled())

    fireEvent.click(screen.getByRole('button', { name: 'статья' }))

    await waitFor(() =>
      expect(searchKnowledgeBase).toHaveBeenLastCalledWith(
        expect.objectContaining({
          query: 'sqlite',
          type: ['article'],
        })
      )
    )
  })
})
