/**
 * @vitest-environment jsdom
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { TooltipProvider } from '@/components/ui/tooltip'
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
      <TooltipProvider>
        <Routes>
          <Route path="/chat" element={<ChatPage />} />
        </Routes>
      </TooltipProvider>
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
    streamChat.mockImplementation((_message, _options, onSources, onToken, onDone) => {
      onSources([
        {
          path: 'articles/sqlite',
          title: 'SQLite',
          type: 'article',
          fragments: [{ heading: 'Intro', snippet: 'sqlite snippet', score: 1, match_type: 'keyword' }],
        },
      ])
      onToken('SQLite answer')
      onDone()
      return new AbortController()
    })
    renderChatPage({ query: 'sqlite' })

    fireEvent.click(screen.getByRole('button', { name: 'Отправить' }))

    expect(screen.getByText('SQLite answer')).toBeInTheDocument()
    expect(screen.getByText('SQLite')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Найденный контекст'))
    expect(screen.getByText('sqlite snippet')).toBeInTheDocument()
  })

  it('renders user and assistant messages as chat bubbles', () => {
    streamChat.mockImplementation((_message, _options, _onSources, onToken, onDone) => {
      onToken('Ответ из базы')
      onDone()
      return new AbortController()
    })
    renderChatPage()

    fireEvent.change(screen.getByPlaceholderText('Спросите что-нибудь...'), {
      target: { value: 'Что есть про sqlite?' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Отправить' }))

    expect(screen.getByText('Что есть про sqlite?')).toBeInTheDocument()
    expect(screen.getByText('Ответ из базы')).toBeInTheDocument()
  })

  it('resets selected search sources', () => {
    renderChatPage({ query: 'sqlite', sourcePaths: ['articles/sqlite'] })

    expect(screen.getByText(/Используются выбранные источники из поиска/)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Сбросить ограничение источников' }))
    fireEvent.click(screen.getByRole('button', { name: 'Отправить' }))

    expect(screen.queryByText(/Используются выбранные источники из поиска/)).not.toBeInTheDocument()
    expect(streamChat).toHaveBeenCalledWith(
      'sqlite',
      { sourcePaths: [] },
      expect.any(Function),
      expect.any(Function),
      expect.any(Function),
      expect.any(Function)
    )
  })
})
