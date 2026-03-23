import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { CheckCircle, FileUp, Loader2, RotateCcw, XCircle } from 'lucide-react'
import {
  acceptImportItem,
  createImportSession,
  getImportSession,
  ingestText,
  rejectImportItem,
  type ImportItem,
} from '../services/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getTypeButtonClass } from '@/lib/type-styles'

const IMPORT_SESSION_KEY = 'kb-telegram-import-session'

type TypeHint = 'auto' | 'article' | 'link' | 'note'

const TYPE_OPTIONS: { value: TypeHint; label: string }[] = [
  { value: 'auto', label: 'Авто' },
  { value: 'article', label: 'Статья' },
  { value: 'link', label: 'Ссылка' },
  { value: 'note', label: 'Заметка' },
]

function ManualAddForm() {
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
    </form>
  )
}

type ImportCompletionSummary = {
  total: number
  processedCount: number
  rejectedCount: number
  /** путь узла, если последнее действие было «Принять» */
  lastSavedPath: string | null
}

function ImportTab() {
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [sessionId, setSessionId] = useState<string | null>(() =>
    localStorage.getItem(IMPORT_SESSION_KEY)
  )
  const [session, setSession] = useState<{
    total: number
    currentIndex: number
    processedCount: number
    rejectedCount: number
    currentItem: ImportItem | null
  } | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [successPath, setSuccessPath] = useState<string | null>(null)
  const [typeHint, setTypeHint] = useState<TypeHint>('auto')
  const [completion, setCompletion] = useState<ImportCompletionSummary | null>(
    null
  )

  useEffect(() => {
    if (sessionId) {
      setLoading(true)
      setError(null)
      getImportSession(sessionId)
        .then((s) => {
          setSession({
            total: s.total,
            currentIndex: s.current_index,
            processedCount: s.processed_count,
            rejectedCount: s.rejected_count,
            currentItem: s.current_item,
          })
        })
        .catch((err) => {
          setError(err instanceof Error ? err.message : 'Ошибка загрузки сессии')
          setSessionId(null)
          localStorage.removeItem(IMPORT_SESSION_KEY)
        })
        .finally(() => setLoading(false))
    } else {
      setSession(null)
    }
  }, [sessionId])

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setError(null)
    setCompletion(null)
    setLoading(true)
    try {
      const json = await file.text()
      const res = await createImportSession(json)
      setSessionId(res.session_id)
      localStorage.setItem(IMPORT_SESSION_KEY, res.session_id)
      setSession({
        total: res.total,
        currentIndex: res.current_index,
        processedCount: 0,
        rejectedCount: 0,
        currentItem: res.current_item,
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка импорта')
    } finally {
      setLoading(false)
      e.target.value = ''
    }
  }

  const handleAccept = async () => {
    if (!sessionId || !session?.currentItem) return
    setError(null)
    setSuccessPath(null)
    setLoading(true)
    try {
      const res = await acceptImportItem(sessionId, typeHint)
      setSuccessPath(res.node.path)
      const nextProcessed = session.processedCount + 1
      setSession({
        total: session.total,
        currentIndex: session.currentIndex + 1,
        processedCount: nextProcessed,
        rejectedCount: session.rejectedCount,
        currentItem: res.next_item,
      })
      if (!res.next_item) {
        setCompletion({
          total: session.total,
          processedCount: nextProcessed,
          rejectedCount: session.rejectedCount,
          lastSavedPath: res.node.path,
        })
        setSessionId(null)
        localStorage.removeItem(IMPORT_SESSION_KEY)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка')
    } finally {
      setLoading(false)
    }
  }

  const handleReject = async () => {
    if (!sessionId || !session?.currentItem) return
    setError(null)
    setSuccessPath(null)
    setLoading(true)
    try {
      const res = await rejectImportItem(sessionId)
      const nextRejected = session.rejectedCount + 1
      setSession({
        total: session.total,
        currentIndex: session.currentIndex + 1,
        processedCount: session.processedCount,
        rejectedCount: nextRejected,
        currentItem: res.next_item,
      })
      if (!res.next_item) {
        setCompletion({
          total: session.total,
          processedCount: session.processedCount,
          rejectedCount: nextRejected,
          lastSavedPath: null,
        })
        setSessionId(null)
        localStorage.removeItem(IMPORT_SESSION_KEY)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка')
    } finally {
      setLoading(false)
    }
  }

  const handleStartOver = () => {
    setSessionId(null)
    localStorage.removeItem(IMPORT_SESSION_KEY)
    setSession(null)
    setError(null)
    setSuccessPath(null)
    setCompletion(null)
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  const dismissCompletion = () => {
    setCompletion(null)
    setSuccessPath(null)
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  if (completion) {
    return (
      <div className="space-y-4">
        <div
          className="rounded-lg border border-green-500/40 bg-green-500/10 px-4 py-5 dark:bg-green-950/40"
          role="status"
        >
          <div className="flex gap-3">
            <CheckCircle className="mt-0.5 h-8 w-8 shrink-0 text-green-600 dark:text-green-400" />
            <div>
              <p className="text-base font-semibold text-foreground">
                Импорт завершён
              </p>
              <p className="mt-1 text-sm text-muted-foreground">
                Все сообщения из файла обработаны. Данные сохранены в базе (для
                принятых записей).
              </p>
              <p className="mt-3 text-sm">
                Всего в выгрузке:{' '}
                <span className="font-medium tabular-nums">{completion.total}</span>
                . Принято:{' '}
                <span className="font-medium tabular-nums text-green-700 dark:text-green-300">
                  {completion.processedCount}
                </span>
                , отклонено:{' '}
                <span className="font-medium tabular-nums">
                  {completion.rejectedCount}
                </span>
                .
              </p>
              {completion.lastSavedPath && (
                <p className="mt-2 text-sm">
                  <Link
                    to={`/node/${completion.lastSavedPath}`}
                    className="font-medium text-green-800 underline hover:no-underline dark:text-green-200"
                  >
                    Открыть последний добавленный узел
                  </Link>
                </p>
              )}
            </div>
          </div>
        </div>
        <Button type="button" onClick={dismissCompletion}>
          Импортировать другой файл
        </Button>
      </div>
    )
  }

  if (!sessionId) {
    return (
      <div className="space-y-4">
        <div className="rounded-lg border-2 border-dashed border-muted-foreground/25 p-8 text-center">
          <input
            ref={fileInputRef}
            type="file"
            accept=".json,application/json"
            onChange={handleFileChange}
            disabled={loading}
            className="hidden"
            id="telegram-import-file"
          />
          <label
            htmlFor="telegram-import-file"
            className="flex cursor-pointer flex-col items-center gap-2 text-muted-foreground hover:text-foreground"
          >
            <FileUp className="h-10 w-10" />
            <span className="text-sm font-medium">
              Выберите JSON-файл экспорта чата Telegram
            </span>
          </label>
        </div>
        {error && (
          <div
            className="flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive"
            role="alert"
          >
            <XCircle className="mt-0.5 h-4 w-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          Прогресс: {session?.currentIndex ?? 0}/{session?.total ?? 0} · Принято:{' '}
          {session?.processedCount ?? 0} · Отклонено: {session?.rejectedCount ?? 0}
        </p>
        <Button variant="outline" size="sm" onClick={handleStartOver}>
          <RotateCcw className="mr-2 h-4 w-4" />
          Начать заново
        </Button>
      </div>

      {loading && !session?.currentItem ? (
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          <span>Загрузка...</span>
        </div>
      ) : session?.currentItem ? (
        <>
          <Card>
            <CardHeader>
              {session.currentItem.source_author && (
                <p className="text-sm text-muted-foreground">
                  {session.currentItem.source_author}
                </p>
              )}
              <CardTitle className="text-base font-normal">
                Текущая запись
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="whitespace-pre-wrap rounded-md bg-muted/50 p-3 text-sm">
                {session.currentItem.text}
              </div>
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
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  onClick={handleReject}
                  disabled={loading}
                >
                  Отклонить
                </Button>
                <Button onClick={handleAccept} disabled={loading}>
                  {loading ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Обработка...
                    </>
                  ) : (
                    'Принять'
                  )}
                </Button>
              </div>
            </CardContent>
          </Card>
        </>
      ) : (
        <p className="text-sm text-muted-foreground">
          Все записи обработаны.
        </p>
      )}

      {error && (
        <div
          className="flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive"
          role="alert"
        >
          <XCircle className="mt-0.5 h-4 w-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}
      {successPath && (
        <div
          className="flex items-start gap-2 rounded-md border border-green-500/50 bg-green-100 px-3 py-2 text-sm text-green-800 dark:bg-green-900/30 dark:text-green-200"
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
    </div>
  )
}

export function AddPage() {
  return (
    <div className="mx-auto max-w-2xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>Добавить</CardTitle>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue="manual" className="w-full">
            <TabsList>
              <TabsTrigger value="manual">Вручную</TabsTrigger>
              <TabsTrigger value="import">Импорт из Telegram</TabsTrigger>
            </TabsList>
            <TabsContent value="manual">
              <ManualAddForm />
            </TabsContent>
            <TabsContent value="import">
              <ImportTab />
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  )
}
