import { useEffect, useRef, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Bot, Send, Square, User, X } from 'lucide-react'
import { streamChat, type ChatSource } from '@/services/api'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { MarkdownContent } from '@/components/MarkdownContent'
import { cn } from '@/lib/utils'

type ChatMessage = {
  id: string
  role: 'user' | 'assistant'
  content: string
  sources?: ChatSource[]
  status?: 'streaming' | 'done' | 'error' | 'stopped'
}

export function ChatPage() {
  const location = useLocation()
  const state = location.state as { query?: string; sourcePaths?: string[] } | null
  const [message, setMessage] = useState(state?.query ?? '')
  const [sourcePaths, setSourcePaths] = useState<string[]>(state?.sourcePaths ?? [])
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const abortRef = useRef<AbortController | null>(null)
  const messagesEndRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (state?.query) setMessage(state.query)
    if (state?.sourcePaths) setSourcePaths(state.sourcePaths)
  }, [state?.query, state?.sourcePaths])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView?.({ block: 'end' })
  }, [messages, loading])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const msg = message.trim()
    if (!msg || loading) return

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
      msg,
      { sourcePaths },
      (srcs) => updateAssistantMessage(assistantID, { sources: srcs }),
      (token) => appendAssistantToken(assistantID, token),
      () => {
        updateAssistantMessage(assistantID, { status: 'done' })
        setLoading(false)
      },
      (err) => {
        updateAssistantMessage(assistantID, { status: 'error' })
        setError(err.message)
        setLoading(false)
      },
    )
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
    <div className="mx-auto flex h-[calc(100dvh-3.5rem)] max-w-4xl flex-col p-4">
      <div
        className={cn(
          'min-h-0 flex-1 space-y-5',
          messages.length > 0 ? 'overflow-y-auto pb-4' : 'overflow-hidden'
        )}
      >
        {messages.length === 0 ? (
          <div className="flex h-full items-center justify-center text-center text-sm text-muted-foreground">
            <div>
              <h1 className="text-xl font-semibold text-foreground">Чат с базой знаний</h1>
              <p className="mt-2">Задайте вопрос, и ответ будет собран по локальным источникам.</p>
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
          <Button type="submit" disabled={!message.trim()} aria-label="Отправить">
            <Send className="size-4" />
          </Button>
        )}
      </form>
    </div>
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
  return (
    <div className="space-y-2 text-xs text-muted-foreground">
      <div>Источники</div>
      <div className="space-y-2">
        {sources.map((source, index) => (
          <div key={`${source.path}-${index}`} className="rounded border bg-muted/20 p-2">
            <Link
              to={`/node/${source.path}`}
              className="font-medium text-primary hover:underline"
            >
              {source.title || source.path}
            </Link>
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
