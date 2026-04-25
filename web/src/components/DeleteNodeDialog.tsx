import { useState } from 'react'
import { deleteNode, type Node } from '@/services/api'
import { useGitStatus } from '@/hooks/useGitStatus'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'

interface DeleteNodeDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  node: Node
  onDeleted: () => void
}

export function DeleteNodeDialog({ open, onOpenChange, node, onDeleted }: DeleteNodeDialogProps) {
  const { refresh: refreshGitStatus } = useGitStatus()
  const [deleting, setDeleting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const meta = node.metadata ?? {}
  const title = (meta.title as string) ?? node.path.split('/').pop() ?? node.path

  const handleDelete = () => {
    setDeleting(true)
    setError(null)
    deleteNode(node.path)
      .then(async () => {
        onOpenChange(false)
        await refreshGitStatus().catch(() => {})
        onDeleted()
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : 'Ошибка удаления')
      })
      .finally(() => setDeleting(false))
  }

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Удалить запись</AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="space-y-2">
              <p>
                Вы уверены, что хотите удалить запись <strong>{title}</strong>?
              </p>
              <p className="font-mono text-xs break-all">{node.path}</p>
              <p>Это действие необратимо.</p>
            </div>
          </AlertDialogDescription>
        </AlertDialogHeader>
        {error && <p className="text-sm text-destructive">{error}</p>}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={deleting}>Отмена</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleDelete}
            disabled={deleting}
            className="bg-destructive text-white hover:bg-destructive/90"
          >
            {deleting ? 'Удаление...' : 'Удалить'}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
