import { useEffect, useRef, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { streamChat, type ChatSource } from '@/services/api'
import { Button } from '@/components/ui/button'

export function ChatPage() {
  const location = useLocation()
  const state = location.state as { query?: string; sourcePaths?: string[] } | null
  const [message, setMessage] = useState(state?.query ?? '')
  const [sourcePaths, setSourcePaths] = useState<string[]>(state?.sourcePaths ?? [])
  const [response, setResponse] = useState('')
  const [sources, setSources] = useState<ChatSource[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const abortRef = useRef<AbortController | null>(null)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!message.trim() || loading) return

    const msg = message
    setResponse('')
    setSources([])
    setError(null)
    setLoading(true)

    abortRef.current = streamChat(
      msg,
      { sourcePaths },
      (srcs) => setSources(srcs),
      (token) => setResponse((prev) => prev + token),
      () => setLoading(false),
      (err) => {
        setError(err.message)
        setLoading(false)
      },
    )
  }

  useEffect(() => {
    if (state?.query) setMessage(state.query)
    if (state?.sourcePaths) setSourcePaths(state.sourcePaths)
  }, [state?.query, state?.sourcePaths])

  const handleStop = () => {
    abortRef.current?.abort()
    setLoading(false)
  }

  return (
    <div className="mx-auto max-w-3xl p-4 space-y-4">
      <h1 className="text-2xl font-bold">Чат с базой знаний</h1>

      {sources.length > 0 && (
        <div className="space-y-1">
          <p className="text-sm text-muted-foreground">Источники:</p>
          <div className="space-y-2">
            {sources.map((s, i) => (
              <div key={i} className="rounded border p-2 text-sm">
                <Link
                  to={`/node/${s.path}`}
                  className="font-medium text-blue-600 hover:underline"
                >
                  {s.title || s.path}
                </Link>
                <span className="ml-2 text-xs text-muted-foreground">{s.type || 'node'}</span>
                {s.fragments && s.fragments.length > 0 && (
                  <details className="mt-2">
                    <summary className="cursor-pointer text-xs text-muted-foreground">Найденный контекст</summary>
                    <div className="mt-2 space-y-2">
                      {s.fragments.map((fragment, idx) => (
                        <div key={idx} className="rounded bg-muted p-2">
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
      )}

      {sourcePaths.length > 0 && sources.length === 0 && (
        <p className="text-sm text-muted-foreground">
          Ответ будет ограничен выбранными источниками: {sourcePaths.join(', ')}
        </p>
      )}

      {response && (
        <div className="whitespace-pre-wrap rounded border p-4 text-sm leading-relaxed">
          {response}
        </div>
      )}

      {error && (
        <p className="text-sm text-destructive">{error}</p>
      )}

      <form onSubmit={handleSubmit} className="flex gap-2">
        <input
          type="text"
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          placeholder="Задайте вопрос о базе знаний..."
          className="flex-1 rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          disabled={loading}
        />
        {loading ? (
          <Button type="button" variant="outline" onClick={handleStop}>
            Стоп
          </Button>
        ) : (
          <Button type="submit" disabled={!message.trim()}>
            Отправить
          </Button>
        )}
      </form>
    </div>
  )
}
