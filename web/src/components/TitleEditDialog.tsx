import { useEffect, useState } from 'react'
import { patchNodeMetadata, type Node } from '@/services/api'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

const titleInputClassName =
  'h-10 rounded-md border border-input bg-background px-3 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2'

interface TitleEditDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  nodePath: string
  initialTitle: string
  onSaved: (node: Node) => void
}

export function TitleEditDialog({
  open,
  onOpenChange,
  nodePath,
  initialTitle,
  onSaved,
}: TitleEditDialogProps) {
  const [draft, setDraft] = useState(initialTitle)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setDraft(initialTitle)
    setError(null)
  }, [open, initialTitle])

  const handleSave = async () => {
    setSaving(true)
    setError(null)
    try {
      const updated = await patchNodeMetadata(nodePath, { title: draft })
      onSaved(updated)
      onOpenChange(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось сохранить')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Редактировать заголовок</DialogTitle>
          <DialogDescription>
            Измените отображаемый title в frontmatter. Пустое значение удалит поле.
          </DialogDescription>
        </DialogHeader>
        <input
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          placeholder="Введите заголовок"
          className={titleInputClassName}
        />
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={saving}>
            Отмена
          </Button>
          <Button onClick={() => void handleSave()} disabled={saving}>
            {saving ? 'Сохранение...' : 'Сохранить'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
