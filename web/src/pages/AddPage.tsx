import { useState } from 'react'
import { Link } from 'react-router-dom'
import { CheckCircle, Loader2, XCircle } from 'lucide-react'
import { ingestText } from '../services/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { getTypeButtonClass } from '@/lib/type-styles'

type TypeHint = 'auto' | 'article' | 'link' | 'note'

const TYPE_OPTIONS: { value: TypeHint; label: string }[] = [
  { value: 'auto', label: 'Авто' },
  { value: 'article', label: 'Статья' },
  { value: 'link', label: 'Ссылка' },
  { value: 'note', label: 'Заметка' },
]

export function AddPage() {
  const [text, setText] = useState('')
  const [typeHint, setTypeHint] = useState<TypeHint>('auto')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [successPath, setSuccessPath] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!text.trim()) return
    setLoading(true)
    setError(null)
    setSuccessPath(null)
    try {
      const node = await ingestText(text.trim(), typeHint)
      setSuccessPath(node.path)
      setText('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="mx-auto max-w-2xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>Добавить</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Тип контента</label>
              <div className="flex gap-1">
                {TYPE_OPTIONS.map(({ value, label }) => (
                  <Button
                    key={value}
                    type="button"
                    variant="outline"
                    size="sm"
                    disabled={loading}
                    className={getTypeButtonClass(value, typeHint === value)}
                    onClick={() => setTypeHint(value)}
                  >
                    {label}
                  </Button>
                ))}
              </div>
              {(typeHint === 'article' || typeHint === 'link') && (
                <p className="text-sm text-muted-foreground">
                  Вставьте URL в текст
                </p>
              )}
            </div>
            <textarea
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="Введите текст..."
              rows={8}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              disabled={loading}
            />
            <Button type="submit" disabled={loading || !text.trim()}>
              {loading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Обработка...
                </>
              ) : (
                'Добавить'
              )}
            </Button>
          </form>
          {error && (
            <div
              className="mt-2 flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive"
              role="alert"
            >
              <XCircle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}
          {successPath && (
            <div
              className="mt-2 flex items-start gap-2 rounded-md border border-green-500/50 bg-green-100 px-3 py-2 text-sm text-green-800 dark:bg-green-900/30 dark:text-green-200"
              role="status"
            >
              <CheckCircle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>
                Добавлено.{' '}
                <Link
                  to={`/node/${successPath}`}
                  className="font-medium underline hover:no-underline"
                >
                  Перейти к узлу
                </Link>
              </span>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
