import { useState } from 'react'
import { Bug } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { createDebugIssue } from '@/services/api'
import { toast } from '@/hooks/use-toast'

type DebugPage = 'node' | 'search' | 'chat'

type Props = {
  page: DebugPage
  title: string
  context: Record<string, unknown>
}

export function DebugIssueDialog({ page, title, context }: Props) {
  const [open, setOpen] = useState(false)
  const [description, setDescription] = useState('')
  const [saving, setSaving] = useState(false)

  const submit = async () => {
    const text = description.trim()
    if (!text) return
    setSaving(true)
    try {
      await createDebugIssue({
        title,
        description: text,
        page,
        context,
      })
      toast({ title: 'Багрепорт сохранён', description: 'Issue создан в локальном debug-хранилище' })
      setDescription('')
      setOpen(false)
    } catch (err) {
      toast({
        title: 'Ошибка',
        description: err instanceof Error ? err.message : 'Не удалось сохранить багрепорт',
        variant: 'destructive',
      })
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button type="button" variant="outline" size="sm">
          <Bug className="mr-2 size-4" />
          Сообщить о проблеме
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Багрепорт</DialogTitle>
          <DialogDescription>
            Опишите проблему. Контекст текущей страницы будет приложен автоматически.
          </DialogDescription>
        </DialogHeader>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          className="min-h-36 w-full rounded border bg-background p-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          placeholder="Что пошло не так? Что ожидалось?"
        />
        <DialogFooter>
          <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
            Отмена
          </Button>
          <Button type="button" onClick={submit} disabled={saving || !description.trim()}>
            {saving ? 'Сохраняем...' : 'Отправить'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

