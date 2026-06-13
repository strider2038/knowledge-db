import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useDebounce } from '@/hooks/useDebounce'
import {
  createNodeAnnotation,
  deleteNodeAnnotation,
  updateNodeAnnotation,
  type NodeAnnotation,
  type NodeAnnotationAnchor,
} from '@/services/api'
import { sortAnnotations } from '@/lib/annotation-anchor'
import { MessageSquarePlus, Trash2 } from 'lucide-react'

type DraftNote = {
  localId: string
  body: string
  anchor: NodeAnnotationAnchor | null
}

export interface NodeAnnotationsPanelProps {
  basePath: string
  content: string
  notes: NodeAnnotation[]
  loading: boolean
  error: string | null
  selectedNoteId: string | null
  pendingAnchor: NodeAnnotationAnchor | null
  onNotesChange: (notes: NodeAnnotation[] | ((prev: NodeAnnotation[]) => NodeAnnotation[])) => void
  onError: (message: string | null) => void
  onSelectNote: (id: string | null) => void
  onClearPendingAnchor: () => void
  onJumpToAnchor: (anchor: NodeAnnotationAnchor) => void
  onReanchorRequest?: (noteId: string) => void
  reanchorNoteId?: string | null
  className?: string
}

function formatAnnotationDate(value: string): string {
  try {
    return new Date(value).toLocaleString()
  } catch {
    return value
  }
}

function NoteEditor({
  note,
  selected,
  saving,
  onSelect,
  onSave,
  onDelete,
  onJumpToAnchor,
  onReanchorRequest,
}: {
  note: NodeAnnotation
  selected: boolean
  saving: boolean
  onSelect: () => void
  onSave: (body: string) => Promise<void>
  onDelete: () => void
  onJumpToAnchor: (anchor: NodeAnnotationAnchor) => void
  onReanchorRequest?: () => void
}) {
  const [body, setBody] = useState(note.body)
  const debouncedBody = useDebounce(body, 500)

  useEffect(() => {
    if (debouncedBody === note.body) return
    void onSave(debouncedBody)
  }, [debouncedBody, note.body, onSave])

  return (
    <article
      className={cn(
        'rounded-lg border bg-card p-3 shadow-sm transition-colors',
        selected && 'border-primary'
      )}
    >
      {note.anchor ? (
        <div className="mb-2 space-y-1">
          <button
            type="button"
            className="line-clamp-2 text-left text-xs text-muted-foreground hover:text-foreground"
            onClick={() => onJumpToAnchor(note.anchor!)}
          >
            «{note.anchor.exact}»
          </button>
          {note.resolved === false ? (
            <div className="space-y-1">
              <p className="text-xs text-amber-700 dark:text-amber-300">Привязка устарела</p>
              {onReanchorRequest ? (
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  className="h-7 text-xs"
                  onClick={onReanchorRequest}
                >
                  Перепривязать
                </Button>
              ) : null}
            </div>
          ) : null}
        </div>
      ) : null}
      <textarea
        value={body}
        onFocus={onSelect}
        onChange={(e) => setBody(e.target.value)}
        rows={4}
        className="w-full resize-y rounded-md border bg-background px-2 py-1.5 text-sm"
      />
      <div className="mt-2 flex items-center justify-between gap-2 text-xs text-muted-foreground">
        <span>{formatAnnotationDate(note.updated)}</span>
        <div className="flex items-center gap-2">
          {saving ? <span>Сохранение...</span> : null}
          <Button
            type="button"
            size="icon"
            variant="ghost"
            className="size-7"
            aria-label="Удалить заметку"
            onClick={onDelete}
          >
            <Trash2 className="size-4" />
          </Button>
        </div>
      </div>
    </article>
  )
}

export function NodeAnnotationsPanel({
  basePath,
  content,
  notes,
  loading,
  error,
  selectedNoteId,
  pendingAnchor,
  onNotesChange,
  onError,
  onSelectNote,
  onClearPendingAnchor,
  onJumpToAnchor,
  onReanchorRequest,
  reanchorNoteId,
  className,
}: NodeAnnotationsPanelProps) {
  const [draft, setDraft] = useState<DraftNote | null>(null)
  const [savingId, setSavingId] = useState<string | null>(null)
  const debouncedDraftBody = useDebounce(draft?.body ?? '', 500)
  const sortedNotes = sortAnnotations(notes, content)

  useEffect(() => {
    if (!pendingAnchor) return
    setDraft({
      localId: `draft-${Date.now()}`,
      body: '',
      anchor: pendingAnchor,
    })
    onSelectNote(null)
    onClearPendingAnchor()
  }, [pendingAnchor, onClearPendingAnchor, onSelectNote])

  useEffect(() => {
    if (!draft || !debouncedDraftBody.trim() || savingId === draft.localId) return
    let cancelled = false
    const run = async () => {
      setSavingId(draft.localId)
      onError(null)
      try {
        const created = await createNodeAnnotation(basePath, {
          body: debouncedDraftBody.trim(),
          anchor: draft.anchor,
        })
        if (cancelled) return
        onNotesChange((prev) => [...prev, created])
        setDraft(null)
        onSelectNote(created.id)
      } catch (err) {
        if (!cancelled) {
          onError(err instanceof Error ? err.message : 'Не удалось сохранить заметку')
        }
      } finally {
        if (!cancelled) setSavingId(null)
      }
    }
    void run()
    return () => {
      cancelled = true
    }
  }, [basePath, debouncedDraftBody, draft, onError, onNotesChange, onSelectNote, savingId])

  const saveExistingNote = async (note: NodeAnnotation, body: string) => {
    if (body === note.body) return
    setSavingId(note.id)
    onError(null)
    try {
      const updated = await updateNodeAnnotation(basePath, note.id, { body })
      onNotesChange((prev) => prev.map((item) => (item.id === updated.id ? updated : item)))
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Не удалось сохранить заметку')
    } finally {
      setSavingId(null)
    }
  }

  const handleDelete = async (note: NodeAnnotation) => {
    onError(null)
    try {
      await deleteNodeAnnotation(basePath, note.id)
      onNotesChange((prev) => prev.filter((item) => item.id !== note.id))
      if (selectedNoteId === note.id) onSelectNote(null)
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Не удалось удалить заметку')
    }
  }

  return (
    <div className={cn('flex h-full flex-col gap-3', className)}>
      <div className="flex items-center justify-between gap-2">
        <h2 className="text-sm font-semibold">Мои заметки</h2>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => {
            setDraft({ localId: `draft-${Date.now()}`, body: '', anchor: null })
            onSelectNote(null)
          }}
        >
          <MessageSquarePlus className="mr-1 size-4" />
          Заметка
        </Button>
      </div>
      {reanchorNoteId ? (
        <p className="text-xs text-muted-foreground">
          Выделите фрагмент в содержании и нажмите «Перепривязать».
        </p>
      ) : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">Загрузка...</p> : null}
      <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto pr-1">
        {draft ? (
          <article className="rounded-lg border bg-card p-3 shadow-sm">
            {draft.anchor ? (
              <button
                type="button"
                className="mb-2 line-clamp-2 text-left text-xs text-muted-foreground hover:text-foreground"
                onClick={() => onJumpToAnchor(draft.anchor!)}
              >
                «{draft.anchor.exact}»
              </button>
            ) : null}
            <textarea
              autoFocus
              value={draft.body}
              onChange={(e) => setDraft({ ...draft, body: e.target.value })}
              rows={4}
              placeholder="Новая заметка..."
              className="w-full resize-y rounded-md border bg-background px-2 py-1.5 text-sm"
            />
            {savingId === draft.localId ? (
              <p className="mt-1 text-xs text-muted-foreground">Сохранение...</p>
            ) : null}
          </article>
        ) : null}
        {sortedNotes.length === 0 && !draft && !loading ? (
          <p className="text-sm text-muted-foreground">Пока нет заметок.</p>
        ) : null}
        {sortedNotes.map((note) => (
          <NoteEditor
            key={note.id}
            note={note}
            selected={selectedNoteId === note.id}
            saving={savingId === note.id}
            onSelect={() => onSelectNote(note.id)}
            onSave={(body) => saveExistingNote(note, body)}
            onDelete={() => void handleDelete(note)}
            onJumpToAnchor={onJumpToAnchor}
            onReanchorRequest={
              onReanchorRequest && note.resolved === false
                ? () => onReanchorRequest(note.id)
                : undefined
            }
          />
        ))}
      </div>
    </div>
  )
}
