/**
 * @vitest-environment jsdom
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { ChatPage } from './ChatPage'

type StreamChat = typeof import('@/services/api')['streamChat']

const { streamChat } = vi.hoisted(() => ({
  streamChat: vi.fn<StreamChat>(() => new AbortController()),
}))

vi.mock('@/services/api', () => ({
  streamChat,
}))

function renderChatPage(state?: { query?: string; sourcePaths?: string[] }) {
  return render(
    <MemoryRouter initialEntries={[{ pathname: '/chat', state }]}>
      <Routes>
        <Route path="/chat" element={<ChatPage />} />
      </Routes>
    </MemoryRouter>
  )
}

describe('ChatPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('uses initial message and source paths from route state', () => {
    renderChatPage({ query: 'sqlite', sourcePaths: ['articles/sqlite'] })

    fireEvent.click(screen.getByRole('button', { name: 'Отправить' }))

    expect(streamChat).toHaveBeenCalledWith(
      'sqlite',
      { sourcePaths: ['articles/sqlite'] },
      expect.any(Function),
      expect.any(Function),
      expect.any(Function),
      expect.any(Function)
    )
  })

  it('renders sources with fragments', () => {
    streamChat.mockImplementation((_message, _options, onSources) => {
      onSources([
        {
          path: 'articles/sqlite',
          title: 'SQLite',
          type: 'article',
          fragments: [{ heading: 'Intro', snippet: 'sqlite snippet', score: 1, match_type: 'keyword' }],
        },
      ])
      return new AbortController()
    })
    renderChatPage({ query: 'sqlite' })

    fireEvent.click(screen.getByRole('button', { name: 'Отправить' }))

    expect(screen.getByText('SQLite')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Найденный контекст'))
    expect(screen.getByText('sqlite snippet')).toBeInTheDocument()
  })
})
