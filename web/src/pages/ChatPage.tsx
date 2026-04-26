import { useState, useRef } from 'react'
import { Link } from 'react-router-dom'
import { streamChat, type ChatSource } from '@/services/api'
import { Button } from '@/components/ui/button'

export function ChatPage() {
  const [message, setMessage] = useState('')
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
      (srcs) => setSources(srcs),
      (token) => setResponse((prev) => prev + token),
      () => setLoading(false),
      (err) => {
        setError(err.message)
        setLoading(false)
      },
    )
  }

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
          <div className="flex flex-wrap gap-2">
            {sources.map((s, i) => (
              <Link
                key={i}
                to={`/node/${s.path}`}
                className="inline-block rounded border px-2 py-1 text-sm text-blue-600 hover:bg-accent"
              >
                {s.title || s.path}
              </Link>
            ))}
          </div>
        </div>
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
