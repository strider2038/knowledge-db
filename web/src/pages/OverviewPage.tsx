import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useLocation, useSearchParams } from 'react-router-dom'
import { ExternalLink } from 'lucide-react'
import {
  getTree,
  getNodesWithParams,
  type TreeNode,
  type NodeListItem,
} from '../services/api'
import { useDebounce } from '../hooks/useDebounce'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import {
  getTypeBadgeColor,
  getTypeButtonClass,
} from '@/lib/type-styles'

const NODE_TYPES = ['article', 'link', 'note'] as const
const DEFAULT_LIMIT = 50

function branchHasMatchingNodes(nodePaths: Set<string>, branchPath: string): boolean {
  for (const p of nodePaths) {
    if (p === branchPath || p.startsWith(branchPath + '/')) return true
  }
  return false
}

function filterTreeByNodePaths(
  node: TreeNode,
  nodePaths: Set<string>
): TreeNode | null {
  if (!node.children?.length) return null
  const filteredChildren: TreeNode[] = []
  for (const child of node.children) {
    const childPath = child.path
    const filtered = filterTreeByNodePaths(child, nodePaths)
    const hasMatch = branchHasMatchingNodes(nodePaths, childPath)
    if (filtered && (filtered.children?.length ?? 0) > 0) {
      filteredChildren.push(filtered)
    } else if (hasMatch) {
      filteredChildren.push(child)
    }
  }
  if (filteredChildren.length === 0) return null
  return { ...node, children: filteredChildren }
}

export function OverviewPage() {
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()
  const path = searchParams.get('path') ?? ''
  const typeFilter = searchParams.get('type')?.split(',').filter(Boolean) ?? []
  const q = searchParams.get('q') ?? ''
  const sort = (searchParams.get('sort') ?? 'title') as
    | 'title'
    | 'type'
    | 'created'
    | 'source_url'
  const order = (searchParams.get('order') ?? 'asc') as 'asc' | 'desc'
  const page = Math.max(1, parseInt(searchParams.get('page') ?? '1', 10))

  const debouncedQ = useDebounce(q, 400)

  const [tree, setTree] = useState<TreeNode | null>(null)
  const [nodes, setNodes] = useState<NodeListItem[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  // Пути узлов по типу для фильтрации дерева (path="" — вся база, без учёта выбранной ветки)
  const [treeFilterPaths, setTreeFilterPaths] = useState<Set<string>>(new Set())

  const updateParams = useCallback(
    (updates: Record<string, string | undefined>) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev)
        for (const [key, value] of Object.entries(updates)) {
          if (value === undefined || value === '') {
            next.delete(key)
          } else {
            next.set(key, value)
          }
        }
        if (next.get('page') === '1') next.delete('page')
        return next
      })
    },
    [setSearchParams]
  )

  useEffect(() => {
    getTree()
      .then(setTree)
      .catch((err) => setError(err instanceof Error ? err.message : 'Ошибка'))
      .finally(() => setLoading(false))
  }, [])

  const typeFilterStr = typeFilter.join(',')
  useEffect(() => {
    let cancelled = false
    queueMicrotask(() => {
      if (!cancelled) setLoading(true)
    })
    const fetchNodes = async () => {
      const { nodes: n, total: t } = await getNodesWithParams({
        path: path || undefined,
        recursive: true,
        q: debouncedQ || undefined,
        type: typeFilter.length > 0 ? typeFilterStr : undefined,
        limit: DEFAULT_LIMIT,
        offset: (page - 1) * DEFAULT_LIMIT,
        sort,
        order,
      })
      if (!cancelled) {
        setNodes(n)
        setTotal(t)
      }
    }
    void fetchNodes()
      .catch(() => {
        if (!cancelled) {
          setNodes([])
          setTotal(0)
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [path, debouncedQ, typeFilterStr, page, sort, order]) // eslint-disable-line react-hooks/exhaustive-deps -- typeFilterStr captures typeFilter

  // При включённом фильтре по типу — загружаем пути по всей базе (path="") для фильтрации дерева.
  // Так дерево не скрывает соседние ветки при выборе конкретного узла.
  useEffect(() => {
    if (typeFilter.length === 0) {
      queueMicrotask(() => setTreeFilterPaths(new Set()))
      return
    }
    let cancelled = false
    getNodesWithParams({
      path: undefined,
      recursive: true,
      type: typeFilterStr,
      limit: 10000,
      offset: 0,
    })
      .then(({ nodes: n }) => {
        if (!cancelled) {
          setTreeFilterPaths(new Set(n.map((x) => x.path)))
        }
      })
      .catch(() => {
        if (!cancelled) setTreeFilterPaths(new Set())
      })
    return () => {
      cancelled = true
    }
  }, [typeFilterStr, typeFilter.length])

  const filteredTree = useMemo(() => {
    if (!tree || treeFilterPaths.size === 0) return tree
    const root = filterTreeByNodePaths(tree, treeFilterPaths)
    return root ?? tree
  }, [tree, treeFilterPaths])

  const toggleType = (t: string) => {
    const next = typeFilter.includes(t)
      ? typeFilter.filter((x) => x !== t)
      : [...typeFilter, t]
    updateParams({ type: next.length > 0 ? next.join(',') : undefined, page: '1' })
  }

  const toggleSort = (field: string) => {
    const nextOrder =
      sort === field && order === 'asc' ? 'desc' : 'asc'
    updateParams({ sort: field, order: nextOrder, page: '1' })
  }

  const totalPages = Math.ceil(total / DEFAULT_LIMIT)

  const renderTree = (node: TreeNode, depth = 0) => {
    const children = [...(node.children ?? [])].sort((a, b) =>
      a.name.localeCompare(b.name, undefined, { sensitivity: 'base' })
    )
    const isRoot = depth === 0
    return (
      <ul
        key={node.path || 'root'}
        className={cn(
          'list-none',
          isRoot ? 'space-y-0.5' : 'ml-4 border-l border-border pl-2 space-y-0.5'
        )}
      >
        {isRoot && (
          <li>
            <button
              type="button"
              onClick={() => updateParams({ path: '', page: '1' })}
              className={cn(
                'block w-full rounded px-2 py-1.5 text-left text-sm font-medium transition-colors hover:bg-accent',
                path === ''
                  ? 'bg-primary/15 text-primary dark:bg-primary/25'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              Вся база
            </button>
          </li>
        )}
        {children.map((child) => {
          const isSelected = path === child.path
          return (
            <li key={child.path}>
              <button
                type="button"
                onClick={() => updateParams({ path: child.path, page: '1' })}
                className={cn(
                  'block w-full rounded px-2 py-1.5 text-left text-sm transition-colors hover:bg-accent',
                  isSelected
                    ? 'bg-primary/15 text-primary font-medium dark:bg-primary/25'
                    : ''
                )}
              >
                {child.name}
              </button>
              {renderTree(child, depth + 1)}
            </li>
          )
        })}
      </ul>
    )
  }

  if (loading && !nodes.length) return <p className="p-4 text-muted-foreground">Загрузка...</p>
  if (error) return <p className="p-4 text-destructive">{error}</p>

  return (
    <div className="flex h-[calc(100vh-3.5rem)]">
      <aside className="w-64 shrink-0 border-r bg-muted/30 p-4 overflow-auto">
        <h3 className="mb-3 font-semibold">Темы</h3>
        {filteredTree && renderTree(filteredTree)}
      </aside>
      <main className="flex-1 overflow-auto p-4">
        <div className="mb-4 flex flex-wrap items-center gap-2">
          <input
            type="search"
            placeholder="Поиск по названию, ключевым словам..."
            value={q}
            onChange={(e) => updateParams({ q: e.target.value || undefined, page: '1' })}
            className="rounded border px-3 py-1.5 text-sm w-64"
          />
          <div className="flex gap-1">
            {NODE_TYPES.map((t) => {
              const isActive = typeFilter.includes(t)
              return (
                <Button
                  key={t}
                  variant="outline"
                  size="sm"
                  className={getTypeButtonClass(t, isActive)}
                  onClick={() => toggleType(t)}
                >
                  {t === 'article' ? 'статья' : t === 'link' ? 'ссылка' : 'заметка'}
                </Button>
              )
            })}
          </div>
        </div>
        <Card>
          <CardHeader>
            <CardTitle>Узлы</CardTitle>
          </CardHeader>
          <CardContent>
            {nodes.length === 0 ? (
              <p className="text-muted-foreground">Нет узлов</p>
            ) : (
              <>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>
                        <button
                          type="button"
                          onClick={() => toggleSort('title')}
                          className="hover:underline"
                        >
                          Название {sort === 'title' && (order === 'asc' ? '↑' : '↓')}
                        </button>
                      </TableHead>
                      <TableHead>
                        <button
                          type="button"
                          onClick={() => toggleSort('type')}
                          className="hover:underline"
                        >
                          Тип {sort === 'type' && (order === 'asc' ? '↑' : '↓')}
                        </button>
                      </TableHead>
                      <TableHead>
                        <button
                          type="button"
                          onClick={() => toggleSort('created')}
                          className="hover:underline"
                        >
                          Дата {sort === 'created' && (order === 'asc' ? '↑' : '↓')}
                        </button>
                      </TableHead>
                      <TableHead>
                        <button
                          type="button"
                          onClick={() => toggleSort('source_url')}
                          className="hover:underline"
                        >
                          Ссылка {sort === 'source_url' && (order === 'asc' ? '↑' : '↓')}
                        </button>
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {nodes.map((n) => (
                      <TableRow key={n.path}>
                        <TableCell>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Link
                                to={`/node/${n.path}`}
                                state={{ returnTo: location.pathname + location.search }}
                                className="text-primary hover:underline"
                              >
                                {n.title || n.path}
                              </Link>
                            </TooltipTrigger>
                            <TooltipContent
                              side="top"
                              className="max-w-sm whitespace-pre-wrap"
                            >
                              {(n.annotation || n.keywords?.length) ? (
                                <>
                                  {n.annotation && (
                                    <p className="mb-1">{n.annotation}</p>
                                  )}
                                  {n.keywords?.length ? (
                                    <p className="text-muted-foreground text-[10px]">
                                      {n.keywords.join(', ')}
                                    </p>
                                  ) : null}
                                </>
                              ) : (
                                <span className="text-muted-foreground">
                                  Нет аннотации
                                </span>
                              )}
                            </TooltipContent>
                          </Tooltip>
                        </TableCell>
                        <TableCell>
                          <span
                            className={cn(
                              'rounded px-1.5 py-0.5 text-xs',
                              getTypeBadgeColor(n.type)
                            )}
                          >
                            {n.type}
                          </span>
                        </TableCell>
                        <TableCell className="text-muted-foreground text-sm">
                          {n.created ? new Date(n.created).toLocaleDateString() : '—'}
                        </TableCell>
                        <TableCell>
                          {(n.type === 'article' || n.type === 'link') && n.source_url ? (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <a
                                  href={n.source_url}
                                  target="_blank"
                                  rel="noopener noreferrer"
                                  className="text-primary hover:text-primary/80 inline-flex"
                                  aria-label={n.source_url}
                                >
                                  <ExternalLink className="size-4" />
                                </a>
                              </TooltipTrigger>
                              <TooltipContent
                                side="top"
                                className="max-w-xs break-all"
                              >
                                {n.source_url}
                              </TooltipContent>
                            </Tooltip>
                          ) : (
                            '—'
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                {totalPages > 1 && (
                  <div className="mt-4 flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={page <= 1}
                      onClick={() => updateParams({ page: String(page - 1) })}
                    >
                      Назад
                    </Button>
                    <span className="text-sm text-muted-foreground">
                      {page} / {totalPages} (всего {total})
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={page >= totalPages}
                      onClick={() => updateParams({ page: String(page + 1) })}
                    >
                      Вперёд
                    </Button>
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>
      </main>
    </div>
  )
}
