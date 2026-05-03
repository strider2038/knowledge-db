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
      meta: { keyword_index: 'fts5' },
    })

    renderSearchPage()

    expect(await screen.findByText('SQLite')).toBeInTheDocument()
    expect(screen.getByText('Local database')).toBeInTheDocument()
    expect(screen.getByText('sqlite snippet')).toBeInTheDocument()
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
