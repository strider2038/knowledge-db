import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getTree, getNodes, type TreeNode } from '../services/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

export function SearchPage() {
  const [tree, setTree] = useState<TreeNode | null>(null)
  const [nodes, setNodes] = useState<TreeNode[]>([])
  const [selectedPath, setSelectedPath] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getTree()
      .then(setTree)
      .catch((err) => setError(err instanceof Error ? err.message : 'Ошибка'))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (!selectedPath) {
      setNodes([])
      return
    }
    getNodes(selectedPath)
      .then(setNodes)
      .catch(() => setNodes([]))
  }, [selectedPath])

  const renderTree = (node: TreeNode, depth = 0) => {
    if (!node.children?.length) return null
    return (
      <ul key={node.path || 'root'} className="ml-4 list-none">
        {node.children.map((child) => (
          <li key={child.path}>
            <button
              type="button"
              onClick={() => setSelectedPath(child.path)}
              className={`w-full rounded px-2 py-1.5 text-left text-sm transition-colors hover:bg-accent ${
                selectedPath === child.path ? 'bg-accent font-medium' : ''
              }`}
            >
              {child.name}
            </button>
            {renderTree(child, depth + 1)}
          </li>
        ))}
      </ul>
    )
  }

  if (loading) return <p className="p-4 text-muted-foreground">Загрузка...</p>
  if (error) return <p className="p-4 text-destructive">{error}</p>

  return (
    <div className="flex h-[calc(100vh-3.5rem)]">
      <aside className="w-64 shrink-0 border-r bg-muted/30 p-4 overflow-auto">
        <h3 className="mb-3 font-semibold">Темы</h3>
        {tree && renderTree(tree)}
      </aside>
      <main className="flex-1 overflow-auto p-4">
        <h3 className="mb-3 font-semibold">Узлы</h3>
        {nodes.length === 0 ? (
          <p className="text-muted-foreground">Выберите тему слева</p>
        ) : (
          <Card>
            <CardHeader>
              <CardTitle>Список узлов</CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Название</TableHead>
                    <TableHead>Путь</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {nodes.map((n) => (
                    <TableRow key={n.path}>
                      <TableCell>
                        <Link
                          to={`/node/${n.path}`}
                          className="text-primary hover:underline"
                        >
                          {n.name}
                        </Link>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {n.path}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        )}
      </main>
    </div>
  )
}
