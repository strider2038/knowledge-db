import { useEffect, useState } from 'react'
import { startNodeAgentEdit } from '@/services/api'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

const instructionStorageKey = (nodePath: string) => `kb:agent-edit-instruction:${nodePath}`

interface NodeAgentEditDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  nodePath: string
  onStarted: (operationId: string) => void
}

export function NodeAgentEditDialog({ open, onOpenChange, nodePath, onStarted }: NodeAgentEditDialogProps) {
  const [instruction, setInstruction] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setError(null)
    try {
      const saved = sessionStorage.getItem(instructionStorageKey(nodePath))
      setInstruction(saved ?? '')
    } catch {
      setInstruction('')
    }
  }, [open, nodePath])

  const trimmed = instruction.trim()
  const canSubmit = trimmed.length > 0 && !submitting

  const handleSubmit = () => {
    if (!canSubmit) return
    setSubmitting(true)
    setError(null)
    startNodeAgentEdit(nodePath, trimmed)
      .then((op) => {
        try {
          sessionStorage.setItem(instructionStorageKey(nodePath), trimmed)
        } catch {
          // ignore storage errors
        }
        onStarted(op.id)
        onOpenChange(false)
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : 'Не удалось запустить редактирование')
      })
      .finally(() => setSubmitting(false))
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Редактировать с агентом</DialogTitle>
          <DialogDescription>
            Опишите, что нужно изменить в узле. Cursor Agent отредактирует файл на сервере; ход выполнения
            отобразится в панели логов.
          </DialogDescription>
        </DialogHeader>
        <textarea
          className="min-h-32 w-full rounded-md border bg-background px-3 py-2 text-sm"
          value={instruction}
          onChange={(e) => setInstruction(e.target.value)}
          placeholder="Например: добавь ключевые слова про Docker и сократи вступление"
          disabled={submitting}
        />
        {error && <p className="text-sm text-destructive">{error}</p>}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>
            Отмена
          </Button>
          <Button type="button" onClick={handleSubmit} disabled={!canSubmit}>
            {submitting ? 'Запуск...' : 'Запустить'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
