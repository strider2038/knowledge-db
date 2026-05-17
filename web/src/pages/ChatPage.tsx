import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Bot, PanelLeft, Pencil, Plus, Send, Square, Trash2, User, X } from 'lucide-react'
import { streamChat, type ChatSource, listChats, createChat, getChat, renameChat, deleteChat, type ChatSession } from '@/services/api'
import { Button } from '@/components/ui/button'
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { MarkdownContent } from '@/components/MarkdownContent'
import { cn } from '@/lib/utils'
import { DebugIssueDialog } from '@/components/DebugIssueDialog'

type ChatMessage = {
  id: string
  role: 'user' | 'assistant'
  content: string
  sources?: ChatSource[]
  status?: 'streaming' | 'done' | 'error' | 'stopped'
}

export function ChatPage() {
  const [sessions, setSessions] = useState<ChatSession[]>([])
  const [activeSessionID, setActiveSessionID] = useState<string>('')
  const [message, setMessage] = useState('')
  const [sourcePaths, setSourcePaths] = useState<string[]>([])
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [loading, setLoading] = useState(false)
  const [bootLoading, setBootLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false)
  const abortRef = useRef<AbortController | null>(null)
  const messagesEndRef = useRef<HTMLDivElement | null>(null)

  const activeSession = useMemo(
    () => sessions.find((session) => session.id === activeSessionID) ?? null,
    [sessions, activeSessionID]
  )

  const openSession = useCallback(async (id: string) => {
    setActiveSessionID(id)
    setError(null)
    try {
      const data = await getChat(id)
      const loadedMessages = Array.isArray(data.messages) ? data.messages : []
      setMessages(
        loadedMessages.map((m) => ({
          id: `loaded-${m.id}`,
          role: m.role,
          content: m.content,
          status: 'done',
        }))
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось открыть чат')
    }
  }, [])

  const initChats = useCallback(async () => {
    setBootLoading(true)
    setError(null)
    try {
      const items = await listChats()
      if (items.length === 0) {
        const created = await createChat()
        setSessions([created])
        setActiveSessionID(created.id)
        setMessages([])
      } else {
        setSessions(items)
        setActiveSessionID(items[0].id)
        await openSession(items[0].id)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить чаты')
    } finally {
      setBootLoading(false)
    }
  }, [openSession])

  useEffect(() => {
    void initChats()
  }, [initChats])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView?.({ block: 'end' })
  }, [messages, loading])

  async function handleCreateChat() {
    try {
      const created = await createChat()
      setSessions((prev) => [created, ...prev])
      setMessages([])
      setActiveSessionID(created.id)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось создать чат')
    }
  }

  async function handleDeleteChat(id: string) {
    if (!confirm('Удалить чат? Это действие необратимо.')) return
    try {
      await deleteChat(id)
      const next = sessions.filter((s) => s.id !== id)
      setSessions(next)
      if (activeSessionID === id) {
        if (next.length === 0) {
          const created = await createChat()
          setSessions([created])
          setActiveSessionID(created.id)
          setMessages([])
        } else {
          await openSession(next[0].id)
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось удалить чат')
    }
  }

  async function handleRenameChat(id: string, currentTitle: string) {
    const title = prompt('Новое название чата', currentTitle)
    if (!title || !title.trim()) return
    try {
      await renameChat(id, title)
      setSessions((prev) => prev.map((s) => (s.id === id ? { ...s, title: title.trim() } : s)))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось переименовать чат')
    }
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const msg = message.trim()
    if (!msg || loading || !activeSessionID) return

    const userMessage: ChatMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content: msg,
    }
    const assistantID = crypto.randomUUID()
    const assistantMessage: ChatMessage = {
      id: assistantID,
      role: 'assistant',
      content: '',
      sources: [],
      status: 'streaming',
    }

    setMessages((prev) => [...prev, userMessage, assistantMessage])
    setMessage('')
    setError(null)
    setLoading(true)

    abortRef.current = streamChat(
      activeSessionID,
      msg,
      { sourcePaths },
      (srcs) => updateAssistantMessage(assistantID, { sources: srcs }),
      (token) => appendAssistantToken(assistantID, token),
      () => {
        updateAssistantMessage(assistantID, { status: 'done' })
        setLoading(false)
        void refreshSessions()
      },
      (err) => {
        updateAssistantMessage(assistantID, { status: 'error' })
        setError(err.message)
        setLoading(false)
      },
    )
  }

  async function refreshSessions() {
    try {
      const items = await listChats()
      setSessions(items)
    } catch {
      // no-op
    }
  }

  const handleStop = () => {
    abortRef.current?.abort()
    setMessages((prev) =>
      prev.map((item) =>
        item.status === 'streaming' ? { ...item, status: 'stopped' } : item
      )
    )
    setLoading(false)
  }

  const updateAssistantMessage = (id: string, patch: Partial<ChatMessage>) => {
    setMessages((prev) =>
      prev.map((item) => (item.id === id ? { ...item, ...patch } : item))
    )
  }

  const appendAssistantToken = (id: string, token: string) => {
    setMessages((prev) =>
      prev.map((item) =>
        item.id === id ? { ...item, content: item.content + token } : item
      )
    )
  }

  return (
    <div className="mx-auto flex h-[calc(100dvh-3.5rem)] w-full max-w-7xl gap-4 p-4">
      <aside className="hidden w-80 shrink-0 rounded-lg border bg-muted/20 p-3 md:flex md:flex-col">
        <ChatSessionsPanel
          sessions={sessions}
          activeSessionID={activeSessionID}
          onCreateChat={handleCreateChat}
          onOpenSession={openSession}
          onRenameChat={handleRenameChat}
          onDeleteChat={handleDeleteChat}
        />
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <div className="mb-2 flex items-center justify-between md:hidden">
          <Sheet open={mobileSidebarOpen} onOpenChange={setMobileSidebarOpen}>
            <SheetTrigger asChild>
              <Button type="button" variant="outline">
                <PanelLeft className="mr-2 size-4" />
                Чаты
              </Button>
            </SheetTrigger>
            <SheetContent side="left" className="w-[88vw] max-w-sm p-3">
              <SheetHeader>
                <SheetTitle>Чаты</SheetTitle>
              </SheetHeader>
              <div className="mt-3 h-[calc(100%-2rem)]">
                <ChatSessionsPanel
                  sessions={sessions}
                  activeSessionID={activeSessionID}
                  onCreateChat={async () => {
                    await handleCreateChat()
                    setMobileSidebarOpen(false)
                  }}
                  onOpenSession={async (id) => {
                    await openSession(id)
                    setMobileSidebarOpen(false)
                  }}
                  onRenameChat={handleRenameChat}
                  onDeleteChat={handleDeleteChat}
                />
              </div>
            </SheetContent>
          </Sheet>
        </div>

        <div
          className={cn(
            'min-h-0 flex-1 space-y-5',
            messages.length > 0 ? 'overflow-y-auto pb-4' : 'overflow-hidden'
          )}
        >
          {bootLoading ? (
            <div className="text-sm text-muted-foreground">Загрузка чатов...</div>
          ) : messages.length === 0 ? (
            <div className="flex h-full items-center justify-center text-center text-sm text-muted-foreground">
              <div>
                <h1 className="text-xl font-semibold text-foreground">Чат с базой знаний</h1>
                <p className="mt-2">{activeSession?.title || 'Новый чат'}</p>
              </div>
            </div>
          ) : (
            messages.map((item) => <MessageBubble key={item.id} message={item} />)
          )}
          <div ref={messagesEndRef} />
        </div>

        {sourcePaths.length > 0 && (
          <div className="mb-2 flex items-center gap-2 rounded border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
            <div className="min-w-0 flex-1 truncate">
              Используются выбранные источники из поиска: {sourcePaths.join(', ')}
            </div>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="size-6 shrink-0"
                  aria-label="Сбросить ограничение источников"
                  onClick={() => setSourcePaths([])}
                >
                  <X className="size-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="top">
                Сбросить ограничение и искать по всей базе
              </TooltipContent>
            </Tooltip>
          </div>
        )}

        {error && <p className="mb-2 text-sm text-destructive">{error}</p>}
        <div className="mb-2">
          <DebugIssueDialog
            page="chat"
            title={`Chat issue: ${activeSession?.title || activeSessionID || 'session'}`}
            context={{
              activeSessionID,
              activeSessionTitle: activeSession?.title || '',
              sourcePaths,
              messages,
            }}
          />
        </div>

        <form onSubmit={handleSubmit} className="flex items-end gap-2 border-t pt-3">
          <textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                e.currentTarget.form?.requestSubmit()
              }
            }}
            placeholder="Спросите что-нибудь..."
            className="max-h-36 min-h-11 flex-1 resize-none rounded border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            disabled={loading}
            rows={1}
          />
          {loading ? (
            <Button type="button" variant="outline" onClick={handleStop} aria-label="Остановить">
              <Square className="size-4" />
            </Button>
          ) : (
            <Button type="submit" disabled={!message.trim() || !activeSessionID} aria-label="Отправить">
              <Send className="size-4" />
            </Button>
          )}
        </form>
      </div>
    </div>
  )
}

function ChatSessionsPanel({
  sessions,
  activeSessionID,
  onCreateChat,
  onOpenSession,
  onRenameChat,
  onDeleteChat,
}: {
  sessions: ChatSession[]
  activeSessionID: string
  onCreateChat: () => void | Promise<void>
  onOpenSession: (id: string) => void | Promise<void>
  onRenameChat: (id: string, title: string) => void | Promise<void>
  onDeleteChat: (id: string) => void | Promise<void>
}) {
  return (
    <>
      <Button type="button" className="mb-3 w-full" onClick={() => void onCreateChat()}>
        <Plus className="mr-2 size-4" /> Новый чат
      </Button>
      <div className="min-h-0 flex-1 space-y-2 overflow-y-auto">
        {sessions.map((session) => (
          <div
            key={session.id}
            className={cn(
              'group rounded border text-sm transition-colors',
              session.id === activeSessionID ? 'border-primary bg-primary/5' : 'bg-background hover:bg-muted/40'
            )}
          >
            <div className="flex items-start gap-2 p-2">
              <button
                type="button"
                className="min-w-0 flex-1 cursor-pointer rounded px-1 py-1 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring"
                onClick={() => void onOpenSession(session.id)}
                aria-label={`Открыть чат: ${session.title || 'Новый чат'}`}
                title="Открыть чат"
              >
                <div className="truncate font-medium">{session.title || 'Новый чат'}</div>
                <div className="text-xs text-muted-foreground">{new Date(session.updated_at).toLocaleString()}</div>
              </button>
              <div className="flex shrink-0 items-center gap-1 md:flex-col xl:flex-row">
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  className="size-7"
                  aria-label={`Переименовать чат: ${session.title || 'Новый чат'}`}
                  title="Переименовать чат"
                  onClick={() => void onRenameChat(session.id, session.title)}
                >
                  <Pencil className="size-3.5" />
                </Button>
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  className="size-7 text-destructive hover:text-destructive"
                  aria-label={`Удалить чат: ${session.title || 'Новый чат'}`}
                  title="Удалить чат"
                  onClick={() => void onDeleteChat(session.id)}
                >
                  <Trash2 className="size-3.5" />
                </Button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </>
  )
}

function MessageBubble({ message }: { message: ChatMessage }) {
  const isUser = message.role === 'user'

  return (
    <article className={cn('flex gap-3', isUser && 'justify-end')}>
      {!isUser && (
        <div className="mt-1 flex size-8 shrink-0 items-center justify-center rounded-full bg-muted">
          <Bot className="size-4" />
        </div>
      )}
      <div className={cn('max-w-[82%] space-y-2', isUser && 'flex flex-col items-end')}>
        <div
          className={cn(
            'rounded-lg px-4 py-3 text-sm leading-relaxed',
            isUser
              ? 'bg-primary text-primary-foreground'
              : 'border bg-background'
          )}
        >
          {message.content ? (
            isUser ? (
              <div className="whitespace-pre-wrap">{message.content}</div>
            ) : (
              <div className="prose prose-sm max-w-none dark:prose-invert">
                <MarkdownContent content={message.content} />
              </div>
            )
          ) : (
            <span className="text-muted-foreground">Думаю...</span>
          )}
          {message.status === 'stopped' && (
            <div className="mt-2 text-xs text-muted-foreground">Остановлено</div>
          )}
        </div>
        {!isUser && message.sources && message.sources.length > 0 && (
          <SourceList sources={message.sources} />
        )}
      </div>
      {isUser && (
        <div className="mt-1 flex size-8 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground">
          <User className="size-4" />
        </div>
      )}
    </article>
  )
}

function SourceList({ sources }: { sources: ChatSource[] }) {
  const nodeHref = (path: string) => {
    const encoded = path.split('/').map(encodeURIComponent).join('/')
    return `/node/${encoded}`
  }

  return (
    <div className="space-y-2 text-xs text-muted-foreground">
      <div>Источники</div>
      <div className="space-y-2">
        {sources.map((source, index) => (
          <div key={`${source.path}-${index}`} className="rounded border bg-muted/20 p-2">
            <a
              className="font-medium text-primary underline-offset-2 hover:underline"
              href={nodeHref(source.path)}
              target="_blank"
              rel="noopener noreferrer"
            >
              {source.title || source.path}
            </a>
            <span className="ml-2">{source.type || 'node'}</span>
            {source.fragments && source.fragments.length > 0 && (
              <details className="mt-2">
                <summary className="cursor-pointer">Найденный контекст</summary>
                <div className="mt-2 space-y-2">
                  {source.fragments.map((fragment, idx) => (
                    <div key={idx} className="rounded bg-background p-2">
                      {fragment.heading && <div className="font-medium">{fragment.heading}</div>}
                      <div>{fragment.snippet || fragment.content}</div>
                    </div>
                  ))}
                </div>
              </details>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
