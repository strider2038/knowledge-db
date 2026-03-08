import { useEffect, useState } from 'react'
import { useLocation, Link, useNavigate } from 'react-router-dom'
import { getNode, type Node } from '../services/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

function translationPath(basePath: string, translationSlug: string): string {
  const lastSlash = basePath.lastIndexOf('/')
  if (lastSlash >= 0) {
    return basePath.slice(0, lastSlash + 1) + translationSlug
  }
  return translationSlug
}

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

  return (
    <div className="mx-auto max-w-3xl space-y-4 p-4">
      <Button variant="ghost" size="sm" asChild>
        <Link to="/">← Назад</Link>
      </Button>
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
      <h2 className="text-2xl font-semibold">{node.path}</h2>
      <Card>
        <CardHeader>
          <CardTitle>Аннотация</CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="whitespace-pre-wrap text-sm">
            {node.annotation || '(нет)'}
          </pre>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Содержание</CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="whitespace-pre-wrap text-sm">
            {node.content || '(нет)'}
          </pre>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Метаданные</CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="overflow-x-auto text-sm">
            {JSON.stringify(node.metadata, null, 2)}
          </pre>
        </CardContent>
      </Card>
    </div>
  )
}
