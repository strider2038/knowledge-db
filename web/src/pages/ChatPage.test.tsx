/**
 * @vitest-environment jsdom
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { TooltipProvider } from '@/components/ui/tooltip'
import { ChatPage } from './ChatPage'

type StreamChat = typeof import('@/services/api')['streamChat']

const {
  streamChat,
  listChats,
  createChat,
  getChat,
  renameChat,
  deleteChat,
} = vi.hoisted(() => ({
  streamChat: vi.fn<StreamChat>(() => new AbortController()),
  listChats: vi.fn(async () => [{ id: 's1', title: 'Chat 1', created_at: new Date().toISOString(), updated_at: new Date().toISOString() }]),
  createChat: vi.fn(async () => ({ id: 's2', title: 'New', created_at: new Date().toISOString(), updated_at: new Date().toISOString() })),
  getChat: vi.fn(async () => ({ session: { id: 's1', title: 'Chat 1', created_at: '', updated_at: '' }, messages: [] })),
  renameChat: vi.fn(async () => {}),
  deleteChat: vi.fn(async () => {}),
}))

vi.mock('@/services/api', () => ({
  streamChat,
  listChats,
  createChat,
  getChat,
  renameChat,
  deleteChat,
}))

function renderChatPage() {
  return render(
    <MemoryRouter initialEntries={[{ pathname: '/chat' }]}>
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

  it('sends message with active session id', async () => {
    renderChatPage()
    await screen.findAllByText('Chat 1')
    expect(screen.getByRole('button', { name: 'Сообщить о проблеме' })).toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText('Спросите что-нибудь...'), {
      target: { value: 'sqlite' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Отправить' }))

    expect(streamChat).toHaveBeenCalledWith(
      's1',
      'sqlite',
      { sourcePaths: [] },
      expect.any(Function),
      expect.any(Function),
      expect.any(Function),
      expect.any(Function)
    )
  })

  it('renders assistant output and sources', async () => {
    streamChat.mockImplementation((_sid, _msg, _options, onSources, onToken, onDone) => {
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

    renderChatPage()
    await screen.findAllByText('Chat 1')

    fireEvent.change(screen.getByPlaceholderText('Спросите что-нибудь...'), {
      target: { value: 'sqlite' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Отправить' }))

    expect(screen.getByText('SQLite answer')).toBeInTheDocument()
    expect(screen.getByText('SQLite')).toBeInTheDocument()
  })

  it('has accessible labels for chat rename and delete actions', async () => {
    renderChatPage()
    await screen.findAllByText('Chat 1')

    expect(
      screen.getByRole('button', { name: 'Переименовать чат: Chat 1' })
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Удалить чат: Chat 1' })
    ).toBeInTheDocument()
  })

  it('handles null messages payload when opening session', async () => {
    getChat.mockResolvedValueOnce({
      session: { id: 's1', title: 'Chat 1', created_at: '', updated_at: '' },
      messages: null,
    } as never)

    renderChatPage()
    const sessionButtons = await screen.findAllByRole('button', { name: /Chat 1/ })
    fireEvent.click(sessionButtons[0])

    expect(await screen.findByText('Чат с базой знаний')).toBeInTheDocument()
  })
})
