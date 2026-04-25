import { useEffect, useState } from 'react'
import { moveNode, getTree, type Node, type TreeNode } from '@/services/api'
import { useGitStatus } from '@/hooks/useGitStatus'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface MoveNodeDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  node: Node
  onMoved: (newPath: string) => void
}

export function MoveNodeDialog({ open, onOpenChange, node, onMoved }: MoveNodeDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        {open ? (
          <MoveNodeDialogInner key={node.path} node={node} onOpenChange={onOpenChange} onMoved={onMoved} />
        ) : null}
      </DialogContent>
    </Dialog>
  )
}

interface MoveNodeDialogInnerProps {
  node: Node
  onOpenChange: (open: boolean) => void
  onMoved: (newPath: string) => void
}

function MoveNodeDialogInner({ node, onOpenChange, onMoved }: MoveNodeDialogInnerProps) {
  const { refresh: refreshGitStatus } = useGitStatus()
  const [targetPath, setTargetPath] = useState(node.path)
  const [tree, setTree] = useState<TreeNode | null>(null)
  const [moving, setMoving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const currentSlug = node.path.split('/').pop() ?? ''

  useEffect(() => {
    let cancelled = false
    getTree()
      .then((t) => {
        if (!cancelled) setTree(t)
      })
      .catch(() => {
        if (!cancelled) setTree(null)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const handleTopicClick = (topicPath: string) => {
    setTargetPath(topicPath ? `${topicPath}/${currentSlug}` : currentSlug)
  }

  const pathChanged = targetPath !== node.path && targetPath.trim() !== ''

  const handleMove = () => {
    if (!pathChanged || moving) return
    setMoving(true)
    setError(null)
    moveNode(node.path, targetPath)
      .then(async () => {
        onOpenChange(false)
        await refreshGitStatus().catch(() => {})
        onMoved(targetPath)
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : 'Ошибка перемещения')
      })
      .finally(() => setMoving(false))
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle>Переместить запись</DialogTitle>
        <DialogDescription asChild>
          <div className="space-y-1">
            <p>Текущий путь:</p>
            <p className="font-mono text-xs break-all text-foreground">{node.path}</p>
          </div>
        </DialogDescription>
      </DialogHeader>
      <div className="space-y-3">
        <div>
          <label className="text-sm font-medium">Новый путь</label>
          <input
            type="text"
            value={targetPath}
            onChange={(e) => setTargetPath(e.target.value)}
            className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            placeholder="topic/subtopic/slug"
          />
        </div>
        {tree && tree.children && tree.children.length > 0 && (
          <div>
            <label className="text-sm font-medium">Выбрать тему</label>
            <div className="mt-1 max-h-48 overflow-y-auto rounded-md border p-2">
              <TopicTreeItems items={tree.children} onSelect={handleTopicClick} />
            </div>
          </div>
        )}
        {error && <p className="text-sm text-destructive">{error}</p>}
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={() => onOpenChange(false)} disabled={moving}>
          Отмена
        </Button>
        <Button onClick={handleMove} disabled={!pathChanged || moving}>
          {moving ? 'Перемещение...' : 'Переместить'}
        </Button>
      </DialogFooter>
    </>
  )
}

function TopicTreeItems({ items, onSelect, depth = 0 }: { items: TreeNode[]; onSelect: (path: string) => void; depth?: number }) {
  return (
    <div className="space-y-0.5">
      {items.map((item) => (
        <div key={item.path}>
          <button
            type="button"
            className="w-full rounded px-2 py-1 text-left text-sm hover:bg-accent"
            style={{ paddingLeft: `${depth * 16 + 8}px` }}
            onClick={() => onSelect(item.path)}
          >
            {item.name}
          </button>
          {item.children && item.children.length > 0 && (
            <TopicTreeItems items={item.children} onSelect={onSelect} depth={depth + 1} />
          )}
        </div>
      ))}
    </div>
  )
}
