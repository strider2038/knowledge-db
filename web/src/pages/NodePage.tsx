import { useEffect, useState } from 'react'
import { useLocation, Link } from 'react-router-dom'
import { getNode, type Node } from '../services/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export function NodePage() {
  const location = useLocation()
  const path = location.pathname.replace(/^\/node\/?/, '')
  const [node, setNode] = useState<Node | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!path) return
    getNode(path)
      .then(setNode)
      .catch((err) => setError(err instanceof Error ? err.message : 'Ошибка'))
      .finally(() => setLoading(false))
  }, [path])

  if (loading) return <p className="p-4 text-muted-foreground">Загрузка...</p>
  if (error) return <p className="p-4 text-destructive">{error}</p>
  if (!node) return null

  return (
    <div className="mx-auto max-w-3xl space-y-4 p-4">
      <Button variant="ghost" size="sm" asChild>
        <Link to="/">← Назад</Link>
      </Button>
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
