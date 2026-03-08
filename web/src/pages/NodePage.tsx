import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { getNode, type Node } from '../services/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { MarkdownContent } from '@/components/MarkdownContent'
import { ContentOutline, ContentOutlineFloating } from '@/components/ContentOutline'
import { extractHeadings } from '@/lib/headings'
import { ExternalLink } from 'lucide-react'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

function translationPath(basePath: string, translationSlug: string): string {
  const lastSlash = basePath.lastIndexOf('/')
  if (lastSlash >= 0) {
    return basePath.slice(0, lastSlash + 1) + translationSlug
  }
  return translationSlug
}

function typeBadgeColor(t: string): string {
  switch (t) {
    case 'article':
      return 'bg-blue-500/20 text-blue-700 dark:text-blue-300'
    case 'link':
      return 'bg-green-500/20 text-green-700 dark:text-green-300'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

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
  '[&_pre]:overflow-x-auto [&_pre]:rounded [&_pre]:p-3'
)

export function NodePage() {
  const location = useLocation()
  const navigate = useNavigate()
  const path = location.pathname.replace(/^\/node\/?/, '')
  const [node, setNode] = useState<Node | null>(null)
  const [originalNode, setOriginalNode] = useState<Node | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const basePath = path.includes('.') ? path.replace(/\.[a-z]{2}$/, '') : path
  const isTranslation = path !== basePath

  useEffect(() => {
    if (!path) return
    getNode(path)
      .then(setNode)
      .catch((err) => setError(err instanceof Error ? err.message : 'Ошибка'))
      .finally(() => setLoading(false))
  }, [path])

  // При просмотре перевода загружаем оригинал, чтобы получить список переводов
  useEffect(() => {
    if (!isTranslation || !basePath) {
      queueMicrotask(() => setOriginalNode(null))
      return
    }
    getNode(basePath)
      .then(setOriginalNode)
      .catch(() => setOriginalNode(null))
  }, [isTranslation, basePath])

  const translations =
    (node?.metadata?.translations as string[] | undefined) ??
    (originalNode?.metadata?.translations as string[] | undefined) ??
    []
  const hasTranslations = translations.length > 0

  if (loading) return <p className="p-4 text-muted-foreground">Загрузка...</p>
  if (error) return <p className="p-4 text-destructive">{error}</p>
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
      {hasTranslations && (
        <div className="flex gap-1 flex-wrap">
          <Button
            variant={!path.includes('.') ? 'default' : 'outline'}
            size="sm"
            onClick={() => navigate(`/node/${basePath}`)}
          >
            Оригинал
          </Button>
          {translations.map((slug) => {
            const transPath = translationPath(basePath, slug)
            const isActive = path === transPath
            const langLabel = slug.includes('.') ? slug.split('.').pop() ?? slug : slug
            return (
              <Button
                key={slug}
                variant={isActive ? 'default' : 'outline'}
                size="sm"
                onClick={() => navigate(`/node/${transPath}`)}
              >
                {langLabel}
              </Button>
            )
          })}
        </div>
      )}
      <h1 className="text-2xl font-semibold">{title}</h1>
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-muted-foreground">
        <span
          className={cn(
            'rounded px-1.5 py-0.5 text-xs',
            typeBadgeColor(nodeType)
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
          <MarkdownContent content={node.annotation || '(нет)'} />
        </CardContent>
      </Card>
      {nodeType !== 'link' && (
        <Card>
          <CardHeader>
            <CardTitle>Содержание</CardTitle>
          </CardHeader>
          <CardContent className={markdownContentClass}>
            <MarkdownContent content={node.content || '(нет)'} />
          </CardContent>
        </Card>
      )}
        </div>
      </div>
    </div>
    </>
  )
}
