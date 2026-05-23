import { useState } from 'react'
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

function loadSavedInstruction(nodePath: string): string {
  try {
    return sessionStorage.getItem(instructionStorageKey(nodePath)) ?? ''
  } catch {
    return ''
  }
}

interface NodeAgentEditDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  nodePath: string
  onStarted: (operationId: string) => void
}

interface NodeAgentEditDialogFormProps {
  nodePath: string
  onOpenChange: (open: boolean) => void
  onStarted: (operationId: string) => void
}

function NodeAgentEditDialogForm({ nodePath, onOpenChange, onStarted }: NodeAgentEditDialogFormProps) {
  const [instruction, setInstruction] = useState(() => loadSavedInstruction(nodePath))
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

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
    <>
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
    </>
  )
}

export function NodeAgentEditDialog({ open, onOpenChange, nodePath, onStarted }: NodeAgentEditDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        {open ? (
          <NodeAgentEditDialogForm
            key={nodePath}
            nodePath={nodePath}
            onOpenChange={onOpenChange}
            onStarted={onStarted}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  )
}
