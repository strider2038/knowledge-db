import { useCallback, useEffect, useState } from 'react'
import {
  getNodeAnnotations,
  updateNodeAnnotation,
  type NodeAnnotation,
  type NodeAnnotationAnchor,
} from '@/services/api'
import { annotationsBasePath } from '@/lib/annotation-anchor'

export function useNodeAnnotations(nodePath: string) {
  const annotationsPath = annotationsBasePath(nodePath)
  const [annotations, setAnnotations] = useState<NodeAnnotation[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedNoteId, setSelectedNoteId] = useState<string | null>(null)
  const [pendingAnchor, setPendingAnchor] = useState<NodeAnnotationAnchor | null>(null)
  const [sheetOpen, setSheetOpen] = useState(false)
  const [reanchorNoteId, setReanchorNoteId] = useState<string | null>(null)

  useEffect(() => {
    if (!annotationsPath) return
    let cancelled = false
    void (async () => {
      setLoading(true)
      setError(null)
      try {
        const data = await getNodeAnnotations(annotationsPath)
        if (!cancelled) setAnnotations(data)
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Не удалось загрузить заметки')
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [annotationsPath])

  const handleReanchorNote = useCallback(
    async (noteId: string, anchor: NodeAnnotationAnchor) => {
      setError(null)
      try {
        const updated = await updateNodeAnnotation(annotationsPath, noteId, { anchor })
        setAnnotations((prev) => prev.map((item) => (item.id === updated.id ? updated : item)))
        setReanchorNoteId(null)
        setSelectedNoteId(updated.id)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Не удалось перепривязать заметку')
      }
    },
    [annotationsPath]
  )

  return {
    annotationsPath,
    annotations,
    setAnnotations,
    loading,
    error,
    setError,
    selectedNoteId,
    setSelectedNoteId,
    pendingAnchor,
    setPendingAnchor,
    sheetOpen,
    setSheetOpen,
    reanchorNoteId,
    setReanchorNoteId,
    handleReanchorNote,
  }
}
