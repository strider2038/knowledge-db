import { useCallback, useRef, useState, type RefObject } from 'react'
import { Button } from '@/components/ui/button'
import { MarkdownContent } from '@/components/MarkdownContent'
import {
  buildAnchorFromSelection,
  isResolvedInContent,
} from '@/lib/annotation-anchor'
import type { NodeAnnotation, NodeAnnotationAnchor } from '@/services/api'
import { cn } from '@/lib/utils'

export interface AnnotatedMarkdownContentProps {
  content: string
  nodePath: string
  contentPath: string
  notes: NodeAnnotation[]
  selectedNoteId: string | null
  onCreateAnchorNote: (anchor: NodeAnnotationAnchor) => void
  onReanchorNote?: (noteId: string, anchor: NodeAnnotationAnchor) => void
  reanchorNoteId?: string | null
  onMarkerClick: (noteId: string) => void
  contentRef?: RefObject<HTMLDivElement | null>
  className?: string
}

export function AnnotatedMarkdownContent({
  content,
  nodePath,
  contentPath,
  notes,
  selectedNoteId,
  onCreateAnchorNote,
  onReanchorNote,
  reanchorNoteId,
  onMarkerClick,
  contentRef,
  className,
}: AnnotatedMarkdownContentProps) {
  const internalRef = useRef<HTMLDivElement>(null)
  const rootRef = contentRef ?? internalRef
  const [selectionText, setSelectionText] = useState('')
  const [toolbarPos, setToolbarPos] = useState<{ top: number; left: number } | null>(null)

  const anchoredNotes = notes.filter(
    (note): note is NodeAnnotation & { anchor: NodeAnnotationAnchor } =>
      !!note.anchor && isResolvedInContent(content, note.anchor)
  )

  const handleMouseUp = useCallback(() => {
    const sel = window.getSelection()
    const text = sel?.toString().trim() ?? ''
    if (!text || !rootRef.current || !sel || sel.rangeCount === 0) {
      setSelectionText('')
      setToolbarPos(null)
      return
    }
    const range = sel.getRangeAt(0)
    if (!rootRef.current.contains(range.commonAncestorContainer)) {
      setSelectionText('')
      setToolbarPos(null)
      return
    }
    const rect = range.getBoundingClientRect()
    const host = rootRef.current.getBoundingClientRect()
    setSelectionText(text)
    setToolbarPos({
      top: rect.bottom - host.top + 8,
      left: Math.max(8, rect.left - host.left),
    })
  }, [rootRef])

  const handleCreateFromSelection = () => {
    if (!selectionText) return
    const anchor = buildAnchorFromSelection(contentPath, content, selectionText)
    if (reanchorNoteId && onReanchorNote) {
      onReanchorNote(reanchorNoteId, anchor)
    } else {
      onCreateAnchorNote(anchor)
    }
    setSelectionText('')
    setToolbarPos(null)
    window.getSelection()?.removeAllRanges()
  }

  const paragraphPrefix = useCallback(
    (text: string) => {
      const markers = anchoredNotes.filter(
        (note) => note.anchor && text.includes(note.anchor.exact)
      )
      if (markers.length === 0) return null
      return (
        <span className="mr-2 inline-flex flex-col gap-0.5 align-top">
          {markers.map((note) => (
            <button
              key={note.id}
              type="button"
              className={cn(
                'text-xs leading-none text-primary hover:text-primary/80',
                selectedNoteId === note.id && 'font-bold'
              )}
              aria-label="Перейти к заметке"
              onClick={() => onMarkerClick(note.id)}
            >
              ●
            </button>
          ))}
        </span>
      )
    },
    [anchoredNotes, onMarkerClick, selectedNoteId]
  )

  return (
    <div ref={rootRef} className="relative" onMouseUp={handleMouseUp}>
      {toolbarPos && selectionText ? (
        <div
          className="absolute z-20"
          style={{ top: toolbarPos.top, left: toolbarPos.left }}
        >
          <Button type="button" size="sm" onClick={handleCreateFromSelection}>
            {reanchorNoteId ? 'Перепривязать' : 'Добавить заметку'}
          </Button>
        </div>
      ) : null}
      <MarkdownContent
        content={content}
        nodePath={nodePath}
        className={className}
        paragraphClassName={(text) =>
          anchoredNotes.some((note) => note.anchor && text.includes(note.anchor.exact))
            ? 'scroll-mt-24'
            : undefined
        }
        paragraphPrefix={paragraphPrefix}
      />
    </div>
  )
}
