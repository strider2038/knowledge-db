import { useEffect, useMemo, useState } from 'react'
import { getKeywordSuggestions, patchNodeMetadata, type Node } from '@/services/api'
import { dedupeKeywords, normalizeKeyword } from '@/lib/keywords'
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

const keywordInputClassName =
  'h-10 flex-1 rounded-md border border-input bg-background px-3 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2'

interface KeywordsEditDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  nodePath: string
  initialKeywords: string[]
  onSaved: (node: Node) => void
}

export function KeywordsEditDialog({
  open,
  onOpenChange,
  nodePath,
  initialKeywords,
  onSaved,
}: KeywordsEditDialogProps) {
  const [draft, setDraft] = useState<string[]>(initialKeywords)
  const [input, setInput] = useState('')
  const [suggestions, setSuggestions] = useState<string[]>([])
  const [loadingSuggestions, setLoadingSuggestions] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setDraft(dedupeKeywords(initialKeywords))
    setInput('')
    setError(null)
  }, [open, initialKeywords])

  useEffect(() => {
    if (!open) return
    if (suggestions.length > 0) return
    let active = true
    setLoadingSuggestions(true)
    getKeywordSuggestions()
      .then((keywords) => {
        if (!active) return
        setSuggestions(keywords)
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
        (keyword) =>
          !draft.some((item) => item.toLocaleLowerCase() === keyword.toLocaleLowerCase())
      )
      .filter((keyword) => (query ? keyword.toLocaleLowerCase().includes(query) : true))
      .slice(0, 8)
  }, [input, suggestions, draft])

  const addKeyword = (rawKeyword: string) => {
    const keyword = normalizeKeyword(rawKeyword)
    if (!keyword) return
    setDraft((prev) => {
      if (prev.some((item) => item.toLocaleLowerCase() === keyword.toLocaleLowerCase())) {
        return prev
      }
      return [...prev, keyword]
    })
    setInput('')
  }

  const handleSave = async () => {
    const normalizedKeywords = dedupeKeywords(draft)
    setSaving(true)
    setError(null)
    try {
      const updated = await patchNodeMetadata(nodePath, { keywords: normalizedKeywords })
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
          <DialogTitle>Редактировать ключевые слова</DialogTitle>
          <DialogDescription>
            Используйте Enter или запятую для добавления тега. Выбирайте из существующих
            ключевиков ниже.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <div className="flex flex-wrap gap-2">
            {draft.length > 0 ? (
              draft.map((keyword) => (
                <span
                  key={keyword}
                  className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-1 text-xs"
                >
                  {keyword}
                  <button
                    type="button"
                    onClick={() => setDraft((prev) => prev.filter((item) => item !== keyword))}
                    aria-label={`Удалить тег ${keyword}`}
                    className="rounded p-0.5 text-muted-foreground hover:text-foreground"
                  >
                    <X className="size-3" />
                  </button>
                </span>
              ))
            ) : (
              <p className="text-xs text-muted-foreground">Пока не добавлено ни одного тега</p>
            )}
          </div>
          <div className="flex gap-2">
            <input
              value={input}
              onChange={(event) => setInput(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ',') {
                  event.preventDefault()
                  addKeyword(input)
                  return
                }
                if (event.key === 'Backspace' && !input.trim()) {
                  setDraft((prev) => prev.slice(0, -1))
                }
              }}
              placeholder="Новый тег"
              className={keywordInputClassName}
            />
            <Button type="button" variant="outline" onClick={() => addKeyword(input)}>
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
                {filteredSuggestions.map((keyword) => (
                  <button
                    key={keyword}
                    type="button"
                    onClick={() => addKeyword(keyword)}
                    className="rounded-full border px-2 py-0.5 text-xs text-foreground transition-colors hover:bg-accent"
                  >
                    {keyword}
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
