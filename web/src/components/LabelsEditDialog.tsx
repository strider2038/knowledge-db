import { useEffect, useMemo, useState } from 'react'
import { getLabelSuggestions, patchNodeMetadata, type Node } from '@/services/api'
import { dedupeLabels, normalizeLabel } from '@/lib/labels'
import { getLabelChipClass } from '@/lib/label-styles'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Plus, X } from 'lucide-react'

const labelInputClassName =
  'h-10 flex-1 rounded-md border border-input bg-background px-3 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2'

interface LabelsEditDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  nodePath: string
  initialLabels: string[]
  onSaved: (node: Node) => void
}

export function LabelsEditDialog({
  open,
  onOpenChange,
  nodePath,
  initialLabels,
  onSaved,
}: LabelsEditDialogProps) {
  const [draft, setDraft] = useState<string[]>(initialLabels)
  const [input, setInput] = useState('')
  const [suggestions, setSuggestions] = useState<string[]>([])
  const [loadingSuggestions, setLoadingSuggestions] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setDraft(dedupeLabels(initialLabels))
    setInput('')
    setError(null)
  }, [open, initialLabels])

  useEffect(() => {
    if (!open) return
    if (suggestions.length > 0) return
    let active = true
    setLoadingSuggestions(true)
    getLabelSuggestions()
      .then((labels) => {
        if (!active) return
        setSuggestions(labels)
      })
      .catch(() => {
        if (!active) return
        setSuggestions([])
      })
      .finally(() => {
        if (active) {
          setLoadingSuggestions(false)
        }
      })
    return () => {
      active = false
    }
  }, [open, suggestions.length])

  const filteredSuggestions = useMemo(() => {
    const query = input.trim().toLocaleLowerCase()
    return suggestions
      .filter(
        (label) =>
          !draft.some((item) => item.toLocaleLowerCase() === label.toLocaleLowerCase())
      )
      .filter((label) => (query ? label.toLocaleLowerCase().includes(query) : true))
      .slice(0, 8)
  }, [input, suggestions, draft])

  const addLabel = (rawLabel: string) => {
    const label = normalizeLabel(rawLabel)
    if (!label || label.includes(',')) return
    setDraft((prev) => {
      if (prev.some((item) => item.toLocaleLowerCase() === label.toLocaleLowerCase())) {
        return prev
      }
      return [...prev, label]
    })
    setInput('')
  }

  const handleSave = async () => {
    const normalizedLabels = dedupeLabels(draft)
    setSaving(true)
    setError(null)
    try {
      const updated = await patchNodeMetadata(nodePath, { labels: normalizedLabels })
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
          <DialogTitle>Редактировать метки</DialogTitle>
          <DialogDescription>
            Личная разметка узла (избранное, «перечитать» и т.д.). Не влияет на семантический
            поиск. Enter или запятая — добавить метку.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <div className="flex flex-wrap gap-2">
            {draft.length > 0 ? (
              draft.map((label) => (
                <span key={label} className={getLabelChipClass(label)}>
                  {label}
                  <button
                    type="button"
                    onClick={() => setDraft((prev) => prev.filter((item) => item !== label))}
                    aria-label={`Удалить метку ${label}`}
                    className="ml-1 rounded p-0.5 opacity-70 hover:opacity-100"
                  >
                    <X className="size-3" />
                  </button>
                </span>
              ))
            ) : (
              <p className="text-xs text-muted-foreground">Меток пока нет</p>
            )}
          </div>
          <div className="flex gap-2">
            <input
              value={input}
              onChange={(event) => setInput(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ',') {
                  event.preventDefault()
                  addLabel(input)
                  return
                }
                if (event.key === 'Backspace' && !input.trim()) {
                  setDraft((prev) => prev.slice(0, -1))
                }
              }}
              placeholder="Новая метка"
              className={labelInputClassName}
            />
            <Button type="button" variant="outline" onClick={() => addLabel(input)}>
              <Plus className="size-4" />
              Добавить
            </Button>
          </div>
          {loadingSuggestions ? (
            <p className="text-xs text-muted-foreground">Загружаем подсказки...</p>
          ) : filteredSuggestions.length > 0 ? (
            <div className="space-y-1">
              <p className="text-xs text-muted-foreground">Подсказки:</p>
              <div className="flex max-h-28 flex-wrap gap-2 overflow-y-auto">
                {filteredSuggestions.map((label) => (
                  <button
                    key={label}
                    type="button"
                    onClick={() => addLabel(label)}
                    className={getLabelChipClass(label)}
                  >
                    {label}
                  </button>
                ))}
              </div>
            </div>
          ) : (
            <p className="text-xs text-muted-foreground">Подсказок нет, можно ввести вручную.</p>
          )}
        </div>
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
