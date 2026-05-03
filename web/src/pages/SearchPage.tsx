import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { ExternalLink, MessageSquare, Search } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { searchKnowledgeBase, type SearchResult } from '@/services/api'
import { getTypeBadgeColor, getTypeButtonClass } from '@/lib/type-styles'
import { cn } from '@/lib/utils'

const NODE_TYPES = ['article', 'link', 'note'] as const
const NODE_TYPE_LABELS: Record<(typeof NODE_TYPES)[number], string> = {
  article: 'статья',
  link: 'ссылка',
  note: 'заметка',
}

export function SearchPage() {
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const query = params.get('q') ?? ''
  const path = params.get('path') ?? ''
  const type = params.get('type')?.split(',').filter(Boolean) ?? []
  const manualProcessed = params.get('manual_processed') ?? ''
  const [draft, setDraft] = useState(query)
  const [results, setResults] = useState<SearchResult[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [searched, setSearched] = useState(false)

  const typeParam = useMemo(() => type.join(','), [type])

  useEffect(() => {
    setDraft(query)
  }, [query])

  useEffect(() => {
    if (!query.trim()) {
      setResults([])
      setSearched(false)
      return
    }
    let cancelled = false
    setLoading(true)
    setError(null)
    setSearched(true)
    void searchKnowledgeBase({
      query,
      type: typeParam ? typeParam.split(',') : undefined,
      path: path || undefined,
      recursive: true,
      manual_processed:
        manualProcessed === 'true'
          ? true
          : manualProcessed === 'false'
            ? false
            : undefined,
      limit: 20,
    })
      .then((res) => {
        if (!cancelled) setResults(res.results)
      })
      .catch((err) => {
        if (!cancelled) {
          setResults([])
          setError(err instanceof Error ? err.message : 'Ошибка поиска')
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [query, typeParam, path, manualProcessed])

  const updateParams = (updates: Record<string, string | undefined>) => {
    setParams((prev) => {
      const next = new URLSearchParams(prev)
      for (const [key, value] of Object.entries(updates)) {
        if (!value) next.delete(key)
        else next.set(key, value)
      }
      return next
    })
  }

  const toggleType = (value: string) => {
    const next = type.includes(value)
      ? type.filter((item) => item !== value)
      : [...type, value]
    updateParams({ type: next.length > 0 ? next.join(',') : undefined })
  }

  const askWithSources = () => {
    navigate('/chat', {
      state: {
        query,
        sourcePaths: results.map((result) => result.path),
      },
    })
  }

  return (
    <div className="mx-auto max-w-5xl space-y-4 p-4">
      <form
        className="flex gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          updateParams({ q: draft.trim() || undefined })
        }}
      >
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          className="min-w-0 flex-1 rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          placeholder="Искать по базе знаний..."
        />
        <Button type="submit" disabled={!draft.trim()}>
          <Search className="mr-2 h-4 w-4" />
          Найти
        </Button>
      </form>

      <div className="flex flex-wrap items-center gap-2 text-sm">
        {NODE_TYPES.map((item) => (
          <Button
            key={item}
            type="button"
            variant="outline"
            size="sm"
            className={getTypeButtonClass(item, type.includes(item))}
            onClick={() => toggleType(item)}
          >
            {NODE_TYPE_LABELS[item]}
          </Button>
        ))}
        <input
          value={path}
          onChange={(e) => updateParams({ path: e.target.value })}
          className="h-9 w-52 rounded border px-2 text-sm"
          placeholder="path/тема"
        />
        <select
          value={manualProcessed}
          onChange={(e) => updateParams({ manual_processed: e.target.value || undefined })}
          className="h-9 rounded border bg-background px-2 text-sm"
        >
          <option value="">Любая проверка</option>
          <option value="true">Проверено</option>
          <option value="false">Не проверено</option>
        </select>
        {results.length > 0 && (
          <Button type="button" variant="outline" size="sm" onClick={askWithSources}>
            <MessageSquare className="mr-2 h-4 w-4" />
            Спросить по этим источникам
          </Button>
        )}
      </div>

      {loading && <p className="text-sm text-muted-foreground">Поиск...</p>}
      {error && <p className="text-sm text-destructive">{error}</p>}
      {!loading && searched && !error && results.length === 0 && (
        <p className="text-sm text-muted-foreground">Ничего не найдено.</p>
      )}

      <div className="space-y-3">
        {results.map((result) => (
          <Card key={result.path} className="gap-0 py-0">
            <CardContent className="p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0 space-y-1">
                  <Link
                    to={`/node/${result.path}`}
                    state={{ returnTo: `/search?${params.toString()}` }}
                    className="text-lg font-semibold leading-snug text-primary hover:underline"
                  >
                    {result.title || result.path}
                  </Link>
                  <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
                    <span
                      className={cn(
                        'rounded px-1.5 py-0.5',
                        getTypeBadgeColor(result.type || 'node')
                      )}
                    >
                      {result.type || 'node'}
                    </span>
                    <span>{result.path}</span>
                  </div>
                </div>
                <span className="rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                  #{result.rank}
                </span>
              </div>

              {result.annotation && (
                <p className="mt-3 text-sm leading-relaxed">{result.annotation}</p>
              )}

              <div className="mt-3 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-muted-foreground">
                {result.keywords?.length > 0 && (
                  <span className="flex flex-wrap gap-1">
                    {result.keywords.map((keyword) => (
                      <span
                        key={keyword}
                        className="rounded-full bg-muted px-2 py-0.5 text-xs"
                      >
                        {keyword}
                      </span>
                    ))}
                  </span>
                )}
                {result.source_url && (
                  <>
                    {result.keywords?.length > 0 && <span>·</span>}
                    <a
                      href={result.source_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex min-w-0 items-center gap-1 text-primary hover:text-primary/80"
                    >
                      <ExternalLink className="size-3.5 shrink-0" />
                      <span className="truncate max-w-[min(28rem,100%)]">
                        {result.source_url}
                      </span>
                    </a>
                  </>
                )}
              </div>

              {(result.match_reasons.length > 0 || result.source_kinds.length > 0) && (
                <div className="mt-3 flex flex-wrap gap-1 text-xs text-muted-foreground">
                  {result.match_reasons.map((reason) => (
                    <span key={reason} className="rounded bg-muted px-2 py-0.5">
                      {reason}
                    </span>
                  ))}
                  {result.source_kinds.map((kind) => (
                    <span key={kind} className="rounded bg-muted px-2 py-0.5">
                      {kind}
                    </span>
                  ))}
                </div>
              )}

              {result.fragments && result.fragments.length > 0 && (
                <div className="mt-3 space-y-2">
                  {result.fragments.map((fragment, idx) => (
                    <div key={idx} className="rounded bg-muted p-3 text-sm">
                      {fragment.heading && <div className="font-medium">{fragment.heading}</div>}
                      <div className="mt-1">{fragment.snippet || fragment.content}</div>
                      <div className="mt-1 text-xs text-muted-foreground">
                        {fragment.match_type}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
