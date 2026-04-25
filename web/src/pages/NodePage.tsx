import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import {
  getNode,
  patchNodeManualProcessed,
  type Node,
} from '../services/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { MarkdownContent } from '@/components/MarkdownContent'
import { ContentOutline, ContentOutlineFloating } from '@/components/ContentOutline'
import { extractHeadings } from '@/lib/headings'
import { ExternalLink, FileQuestion } from 'lucide-react'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import { getTypeBadgeColor } from '@/lib/type-styles'
import { NodeActionBar } from '@/components/NodeActionBar'
import { useGitStatus } from '@/hooks/useGitStatus'

function formatDate(value: unknown): string {
  if (!value || typeof value !== 'string') return '—'
  try {
    return new Date(value).toLocaleDateString()
  } catch {
    return '—'
  }
}

const markdownContentClass = cn(
  'prose prose-base dark:prose-invert max-w-none',
  'prose-h1:mt-8 prose-h1:mb-6 prose-h1:text-2xl prose-h1:font-bold',
  'prose-h2:mt-10 prose-h2:mb-4 prose-h2:text-xl prose-h2:font-bold prose-h2:border-b prose-h2:border-border prose-h2:pb-2',
  'prose-h3:mt-8 prose-h3:mb-3 prose-h3:text-lg prose-h3:font-semibold',
  'prose-h4:mt-6 prose-h4:mb-2 prose-h4:text-base prose-h4:font-semibold',
  'prose-p:my-4 prose-p:leading-7',
  'prose-ul:my-4 prose-ol:my-4 prose-li:my-1',
  'prose-blockquote:my-4 prose-blockquote:border-l-4 prose-blockquote:border-primary prose-blockquote:pl-4 prose-blockquote:italic',
  '[&_pre]:my-4',
  // Inline formula images (inside paragraphs with text) — как на Хабре
  '[&_p_img]:inline [&_p_img]:align-middle [&_p_img]:my-0',
  // Standalone images (paragraph with only img) — оставляем block
  '[&_p:has(>img:only-child)_img]:block [&_p:has(>img:only-child)_img]:my-4'
)

export function NodePage() {
  const location = useLocation()
  const navigate = useNavigate()
  const { refresh: refreshGitStatus } = useGitStatus()
  const path = location.pathname.replace(/^\/node\/?/, '')
  const [node, setNode] = useState<Node | null>(null)
  const [originalNode, setOriginalNode] = useState<Node | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [manualSaving, setManualSaving] = useState(false)
  const [manualError, setManualError] = useState<string | null>(null)

  const basePath = path.includes('.') ? path.replace(/\.[a-z]{2}$/, '') : path
  const isTranslation = path !== basePath

  useEffect(() => {
    if (!path) return
    getNode(path)
      .then(setNode)
      .catch((err) => setError(err instanceof Error ? err.message : 'Ошибка'))
      .finally(() => setLoading(false))
  }, [path])

  useEffect(() => {
    if (!isTranslation || !basePath) {
      queueMicrotask(() => setOriginalNode(null))
      return
    }
    getNode(basePath)
      .then(setOriginalNode)
      .catch(() => setOriginalNode(null))
  }, [isTranslation, basePath])

  const handleManualProcessedToggle = async (next: boolean) => {
    if (!node) return
    setManualError(null)
    const prev = node
    setManualSaving(true)
    setNode({
      ...node,
      metadata: { ...node.metadata, manual_processed: next },
    })
    try {
      const updated = await patchNodeManualProcessed(node.path, next)
      setNode(updated)
      await refreshGitStatus().catch(() => {})
    } catch (err) {
      setNode(prev)
      setManualError(err instanceof Error ? err.message : 'Не удалось сохранить')
    } finally {
      setManualSaving(false)
    }
  }

  const handleNodeChanged = () => {
    if (!path) return
    getNode(path).then(setNode)
    if (basePath && isTranslation) {
      getNode(basePath).then(setOriginalNode)
    }
  }

  const translations =
    (node?.metadata?.translations as string[] | undefined) ??
    (originalNode?.metadata?.translations as string[] | undefined) ??
    []
  const hasTranslations = translations.length > 0

  if (loading) return <p className="p-4 text-muted-foreground">Загрузка...</p>
  if (error)
    return (
      <div className="flex gap-8 p-4 lg:px-8">
        <div className="min-w-0 flex-1">
          <div className="mx-auto max-w-3xl space-y-4">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigate(location.state?.returnTo ?? '/')}
            >
              ← Назад
            </Button>
            <nav className="flex items-center gap-1 text-sm text-muted-foreground">
              <Link to="/" className="hover:text-foreground">
                Обзор
              </Link>
            </nav>
            <Card className="border-dashed">
              <CardContent className="flex flex-col items-center justify-center py-16 text-center">
                <FileQuestion className="mb-4 size-16 text-muted-foreground/60" />
                <h2 className="mb-2 text-xl font-semibold">Запись не найдена</h2>
                <p className={cn('max-w-sm text-muted-foreground', error ? 'mb-2' : 'mb-6')}>
                  {path ? (
                    <>Запись по пути «{path}» не существует или недоступна.</>
                  ) : (
                    <>Указанный путь пуст или некорректен.</>
                  )}
                </p>
                {error && (
                  <p className="mb-6 text-sm text-muted-foreground/80">{error}</p>
                )}
                <div className="flex flex-wrap justify-center gap-2">
                  <Button
                    variant="outline"
                    onClick={() => navigate(location.state?.returnTo ?? '/')}
                  >
                    Назад
                  </Button>
                  <Button asChild>
                    <Link to="/">К обзору</Link>
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    )
  if (!node) return null

  const meta = node.metadata ?? {}
  const nodeType = (meta.type as string) ?? 'note'
  const title = (meta.title as string) ?? basePath.split('/').pop() ?? basePath
  const created = formatDate(meta.created)
  const updated = formatDate(meta.updated)
  const sourceUrl = meta.source_url as string | undefined
  const sourceAuthor = meta.source_author as string | undefined
  const sourceDate = meta.source_date as string | undefined
  const keywords = (meta.keywords as string[] | undefined) ?? []
  const hasSourceAttribution = !!(sourceUrl || sourceAuthor || sourceDate)

  const segments = basePath.split('/').filter(Boolean)
  const breadcrumbLinks = segments.map((seg, i) => {
    const path = segments.slice(0, i + 1).join('/')
    return { name: seg, path }
  })

  const hasOutline =
    nodeType !== 'link' && extractHeadings(node.content || '').length > 0

  return (
    <>
      {hasOutline && <ContentOutlineFloating content={node.content || ''} />}
      <div className="flex gap-8 p-4 lg:px-8">
      {hasOutline && (
        <aside className="hidden w-56 shrink-0 lg:block">
          <ContentOutline content={node.content || ''} />
        </aside>
      )}
      <div className="min-w-0 flex-1">
        <div className="mx-auto max-w-3xl space-y-4">
      <Button
        variant="ghost"
        size="sm"
        onClick={() => navigate(location.state?.returnTo ?? '/')}
      >
        ← Назад
      </Button>
      <nav className="flex items-center gap-1 text-sm text-muted-foreground">
        <Link to="/" className="hover:text-foreground">
          Обзор
        </Link>
        {breadcrumbLinks.map(({ name, path: segPath }) => (
          <span key={segPath} className="flex items-center gap-1">
            <span>›</span>
            <Link to={`/?path=${encodeURIComponent(segPath)}`} className="hover:text-foreground">
              {name}
            </Link>
          </span>
        ))}
      </nav>
      {node && (
        <NodeActionBar
          node={node}
          basePath={basePath}
          currentNodePath={path}
          isTranslation={isTranslation}
          hasTranslations={hasTranslations}
          translations={translations}
          manualProcessed={!!(node.metadata?.manual_processed)}
          manualSaving={manualSaving}
          onManualProcessedToggle={handleManualProcessedToggle}
          onNodeChanged={handleNodeChanged}
          onNavigate={(p) => navigate(p)}
          onNavigateHome={() => navigate('/')}
        />
      )}
      <div className="flex flex-wrap items-start gap-x-3 gap-y-2">
        <h1 className="text-2xl font-semibold leading-snug">{title}</h1>
      </div>
      {manualError ? (
        <p className="text-sm text-destructive">{manualError}</p>
      ) : null}
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-muted-foreground">
        <span
          className={cn(
            'rounded px-1.5 py-0.5 text-xs',
            getTypeBadgeColor(nodeType)
          )}
        >
          {nodeType}
        </span>
        <span>{created}</span>
        <span>·</span>
        <span>{updated}</span>
        {keywords.length > 0 && (
          <>
            <span>·</span>
            <span className="flex flex-wrap gap-1">
              {keywords.map((kw) => (
                <span
                  key={kw}
                  className="rounded-full bg-muted px-2 py-0.5 text-xs"
                >
                  {kw}
                </span>
              ))}
            </span>
          </>
        )}
      </div>
      {hasSourceAttribution && (
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-muted-foreground">
          {sourceUrl && (
            <Tooltip>
              <TooltipTrigger asChild>
                <a
                  href={sourceUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 text-primary hover:text-primary/80"
                  aria-label={sourceUrl}
                >
                  <ExternalLink className="size-4 shrink-0" />
                  <span className="truncate max-w-[min(24rem,100%)]">{sourceUrl}</span>
                </a>
              </TooltipTrigger>
              <TooltipContent side="top" className="max-w-xs break-all">
                {sourceUrl}
              </TooltipContent>
            </Tooltip>
          )}
          {sourceAuthor && (
            <>
              {sourceUrl && <span>·</span>}
              <span>Автор: {sourceAuthor}</span>
            </>
          )}
          {sourceDate && (
            <>
              {(sourceUrl || sourceAuthor) && <span>·</span>}
              <span>Дата источника: {formatDate(sourceDate)}</span>
            </>
          )}
        </div>
      )}
      {nodeType === 'link' && sourceUrl && (
        <a
          href={sourceUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-2 rounded-lg border bg-card px-4 py-3 text-card-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
        >
          <ExternalLink className="size-4 shrink-0" />
          <span className="truncate">{sourceUrl}</span>
        </a>
      )}
      <Card>
        <CardHeader>
          <CardTitle>Аннотация</CardTitle>
        </CardHeader>
        <CardContent className={markdownContentClass}>
          <MarkdownContent content={node.annotation || '(нет)'} nodePath={node.path} />
        </CardContent>
      </Card>
      {nodeType !== 'link' && (
        <Card>
          <CardHeader>
            <CardTitle>Содержание</CardTitle>
          </CardHeader>
          <CardContent className={markdownContentClass}>
            <MarkdownContent content={node.content || '(нет)'} nodePath={node.path} />
          </CardContent>
        </Card>
      )}
        </div>
      </div>
    </div>
    </>
  )
}
