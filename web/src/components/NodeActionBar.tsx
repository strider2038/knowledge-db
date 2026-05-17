import { useEffect, useRef, useState } from 'react'
import { Check, ChevronDown, ChevronUp, Languages, Move, RefreshCw, Sparkles, Trash2 } from 'lucide-react'
import {
  type Node,
  type NodeDumpImagesLogEntry,
  type NodeNormalizationLogEntry,
  getNodeDumpImagesLogs,
  getNodeDumpImagesStatus,
  getJobStatus,
  getNode,
  getNodeNormalizationLogs,
  getNodeNormalizationStatus,
  startJob,
  startNodeDumpImages,
  startNodeNormalization,
} from '@/services/api'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { DeleteNodeDialog } from '@/components/DeleteNodeDialog'
import { MoveNodeDialog } from '@/components/MoveNodeDialog'
import { MarkdownContent } from '@/components/MarkdownContent'

function translationPath(basePath: string, translationSlug: string): string {
  const lastSlash = basePath.lastIndexOf('/')
  if (lastSlash >= 0) {
    return basePath.slice(0, lastSlash + 1) + translationSlug
  }
  return translationSlug
}

interface NodeActionBarProps {
  node: Node
  basePath: string
  currentNodePath: string
  isTranslation: boolean
  hasTranslations: boolean
  translations: string[]
  manualProcessed: boolean
  manualSaving: boolean
  onManualProcessedToggle: (value: boolean) => void
  onNodeChanged: (node: Node) => void
  onNavigate: (path: string) => void
  onNavigateHome: () => void
}

export function NodeActionBar({
  node,
  basePath,
  currentNodePath,
  isTranslation,
  hasTranslations,
  translations,
  manualProcessed,
  manualSaving,
  onManualProcessedToggle,
  onNodeChanged,
  onNavigate,
  onNavigateHome,
}: NodeActionBarProps) {
  const [translating, setTranslating] = useState(false)
  const [translateError, setTranslateError] = useState<string | null>(null)
  const [refreshing, setRefreshing] = useState(false)
  const [refreshError, setRefreshError] = useState<string | null>(null)
  const [refreshSuccess, setRefreshSuccess] = useState(false)
  const [normalizing, setNormalizing] = useState(false)
  const [normalizeError, setNormalizeError] = useState<string | null>(null)
  const [normalizeSuccess, setNormalizeSuccess] = useState(false)
  const [normalizeOpID, setNormalizeOpID] = useState<string | null>(null)
  const [normalizeStage, setNormalizeStage] = useState<string>('')
  const [normalizeFinished, setNormalizeFinished] = useState(false)
  const [logsPanelOpen, setLogsPanelOpen] = useState(false)
  const [logViewMode, setLogViewMode] = useState<'compact' | 'raw'>('compact')
  const [logEntries, setLogEntries] = useState<NodeNormalizationLogEntry[]>([])
  const logOffsetRef = useRef(0)
  const logsContainerRef = useRef<HTMLDivElement | null>(null)
  const [dumpingImages, setDumpingImages] = useState(false)
  const [dumpImagesError, setDumpImagesError] = useState<string | null>(null)
  const [dumpImagesSuccess, setDumpImagesSuccess] = useState(false)
  const [dumpImagesOpID, setDumpImagesOpID] = useState<string | null>(null)
  const [dumpImagesStage, setDumpImagesStage] = useState<string>('')
  const [dumpImagesFinished, setDumpImagesFinished] = useState(false)
  const [dumpLogsPanelOpen, setDumpLogsPanelOpen] = useState(false)
  const [dumpLogEntries, setDumpLogEntries] = useState<NodeDumpImagesLogEntry[]>([])
  const dumpLogOffsetRef = useRef(0)
  const dumpLogsContainerRef = useRef<HTMLDivElement | null>(null)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [moveOpen, setMoveOpen] = useState(false)

  const meta = node.metadata ?? {}
  const nodeType = (meta.type as string) ?? 'note'
  const canTranslate = nodeType === 'article' && !hasTranslations && !isTranslation
  const canDumpImages = !isTranslation && nodeType === 'article'
  const canRefreshDescription = !isTranslation && typeof meta.source_url === 'string' && meta.source_url.length > 0
  const showManualProcessed = !isTranslation
  const showTranslate = canTranslate

  useEffect(() => {
    if (!normalizeOpID || normalizeFinished) return
    const tick = () => {
      getNodeNormalizationStatus(normalizeOpID)
        .then((status) => {
          setNormalizeStage(status.stage)
          if (status.status === 'running') {
            setNormalizing(true)
          } else {
            setNormalizing(false)
            setNormalizeFinished(true)
            if (status.status === 'error') {
              setNormalizeError(status.error ?? 'Ошибка нормализации')
            } else {
              setNormalizeSuccess(true)
              onNodeChanged(node)
            }
          }
        })
        .catch((err) => setNormalizeError(err instanceof Error ? err.message : 'Ошибка статуса нормализации'))

      getNodeNormalizationLogs(normalizeOpID, logOffsetRef.current)
        .then((resp) => {
          if (resp.entries.length > 0) {
            setLogEntries((prev) => [...prev, ...resp.entries])
          }
          logOffsetRef.current = resp.next_offset
        })
        .catch(() => {})
    }

    tick()
    const interval = setInterval(tick, 1500)
    return () => clearInterval(interval)
  }, [normalizeOpID, normalizeFinished, onNodeChanged, node])

  useEffect(() => {
    if (!dumpImagesOpID || dumpImagesFinished) return
    const tick = () => {
      getNodeDumpImagesStatus(dumpImagesOpID)
        .then((status) => {
          setDumpImagesStage(status.stage)
          if (status.status === 'running') {
            setDumpingImages(true)
          } else {
            setDumpingImages(false)
            setDumpImagesFinished(true)
            if (status.status === 'error') {
              setDumpImagesError(status.error ?? 'Ошибка выгрузки изображений')
            } else {
              setDumpImagesSuccess(true)
              onNodeChanged(node)
            }
          }
        })
        .catch((err) => setDumpImagesError(err instanceof Error ? err.message : 'Ошибка статуса выгрузки изображений'))

      getNodeDumpImagesLogs(dumpImagesOpID, dumpLogOffsetRef.current)
        .then((resp) => {
          if (resp.entries.length > 0) {
            setDumpLogEntries((prev) => [...prev, ...resp.entries])
          }
          dumpLogOffsetRef.current = resp.next_offset
        })
        .catch(() => {})
    }

    tick()
    const interval = setInterval(tick, 1500)
    return () => clearInterval(interval)
  }, [dumpImagesOpID, dumpImagesFinished, onNodeChanged, node])

  useEffect(() => {
    if (!logsPanelOpen || !logsContainerRef.current) return
    logsContainerRef.current.scrollTop = logsContainerRef.current.scrollHeight
  }, [logEntries, logsPanelOpen])

  useEffect(() => {
    if (!dumpLogsPanelOpen || !dumpLogsContainerRef.current) return
    dumpLogsContainerRef.current.scrollTop = dumpLogsContainerRef.current.scrollHeight
  }, [dumpLogEntries, dumpLogsPanelOpen])

  const handleTranslate = () => {
    setTranslating(true)
    setTranslateError(null)
    startJob('article_translate', basePath)
      .then((job) => {
        const interval = setInterval(() => {
          getJobStatus(job.id).then((s) => {
            if (s.status === 'success' || s.status === 'error') {
              clearInterval(interval)
              setTranslating(false)
              if (s.status === 'error') setTranslateError(s.error ?? 'Ошибка перевода')
              onNodeChanged(node)
            }
          }).catch(() => {
            clearInterval(interval)
            setTranslating(false)
            setTranslateError('Ошибка при проверке статуса')
          })
        }, 1500)
      })
      .catch((err) => {
        setTranslating(false)
        setTranslateError(err instanceof Error ? err.message : 'Ошибка перевода')
      })
  }

  const handleMoved = (newPath: string) => {
    onNavigate(`/node/${newPath}`)
  }

  const handleRefreshDescription = () => {
    setRefreshing(true)
    setRefreshError(null)
    setRefreshSuccess(false)
    startJob('refresh_description', basePath)
      .then((job) => {
        const interval = setInterval(() => {
          getJobStatus(job.id).then(async (s) => {
            if (s.status === 'success' || s.status === 'error') {
              clearInterval(interval)
              setRefreshing(false)
              if (s.status === 'error') {
                setRefreshError(s.error ?? 'Не удалось обновить описание')
                return
              }
              setRefreshSuccess(true)
              const updated = await getNode(basePath)
              onNodeChanged(updated)
            }
          }).catch(() => {
            clearInterval(interval)
            setRefreshing(false)
            setRefreshError('Не удалось обновить описание')
          })
        }, 1500)
      })
      .catch((err) => {
        setRefreshing(false)
        setRefreshError(err instanceof Error ? err.message : 'Не удалось обновить описание')
      })
  }

  const showPrimaryGroup = showManualProcessed || showTranslate || hasTranslations || canRefreshDescription

  const handleNormalize = () => {
    setNormalizing(true)
    setNormalizeError(null)
    setNormalizeSuccess(false)
    setNormalizeFinished(false)
    setLogsPanelOpen(true)
    setLogEntries([])
    logOffsetRef.current = 0
    startNodeNormalization(basePath)
      .then((op) => {
        setNormalizeOpID(op.id)
        setNormalizeStage(op.stage)
      })
      .catch(async () => {
        const job = await startJob('node_normalize', basePath)
        setNormalizeOpID(job.id)
        setNormalizeStage(job.stage)
      })
      .catch((err) => {
        setNormalizing(false)
        setNormalizeError(err instanceof Error ? err.message : 'Не удалось запустить нормализацию')
      })
  }

  const handleDumpImages = () => {
    setDumpingImages(true)
    setDumpImagesError(null)
    setDumpImagesSuccess(false)
    setDumpImagesFinished(false)
    setDumpLogsPanelOpen(true)
    setDumpLogEntries([])
    dumpLogOffsetRef.current = 0
    startNodeDumpImages(basePath)
      .then((op) => {
        setDumpImagesOpID(op.id)
        setDumpImagesStage(op.stage)
      })
      .catch(async () => {
        const job = await startJob('node_dump_images', basePath)
        setDumpImagesOpID(job.id)
        setDumpImagesStage(job.stage)
      })
      .catch((err) => {
        setDumpingImages(false)
        setDumpImagesError(err instanceof Error ? err.message : 'Не удалось запустить выгрузку изображений')
      })
  }

  return (
    <>
      <div className="sticky top-0 z-10 -mx-4 border-b bg-background/95 px-4 py-2 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="flex flex-wrap items-center gap-2">
          {showManualProcessed && (
            <Button
              type="button"
              variant={manualProcessed ? 'ghost' : 'outline'}
              size="sm"
              className="h-8"
              disabled={manualSaving}
              onClick={() => onManualProcessedToggle(!manualProcessed)}
            >
              <Check className={cn('mr-1 size-4', manualProcessed && 'text-green-600 dark:text-green-400')} />
              Проверено
            </Button>
          )}

          {showTranslate && (
            <Button variant="outline" size="sm" className="h-8" onClick={handleTranslate} disabled={translating}>
              <Languages className="mr-1 size-4" />
              {translating ? 'Перевод...' : 'Перевести'}
            </Button>
          )}

          {hasTranslations && (
            <div className="flex flex-wrap gap-1">
              <Button variant={currentNodePath === basePath ? 'default' : 'outline'} size="sm" className="h-8" onClick={() => onNavigate(`/node/${basePath}`)}>
                Оригинал
              </Button>
              {translations.map((slug) => {
                const transPath = translationPath(basePath, slug)
                const isActive = currentNodePath === transPath
                const langLabel = slug.includes('.') ? slug.split('.').pop() ?? slug : slug
                return (
                  <Button key={slug} variant={isActive ? 'default' : 'outline'} size="sm" className="h-8" onClick={() => onNavigate(`/node/${transPath}`)}>
                    {langLabel}
                  </Button>
                )
              })}
            </div>
          )}

          {!isTranslation && (
            <Button variant="outline" size="sm" className="h-8" onClick={handleNormalize} disabled={normalizing}>
              <Sparkles className={cn('mr-1 size-4', normalizing && 'animate-pulse')} />
              {normalizing ? 'Нормализация...' : 'Нормализация'}
            </Button>
          )}

          {canDumpImages && (
            <Button variant="outline" size="sm" className="h-8" onClick={handleDumpImages} disabled={dumpingImages}>
              <Sparkles className={cn('mr-1 size-4', dumpingImages && 'animate-pulse')} />
              {dumpingImages ? 'Выгрузка изображений...' : 'Выгрузить изображения'}
            </Button>
          )}

          {canRefreshDescription && (
            <Button variant="outline" size="sm" className="h-8" onClick={handleRefreshDescription} disabled={refreshing}>
              <RefreshCw className={cn('mr-1 size-4', refreshing && 'animate-spin')} />
              {refreshing ? 'Обновление...' : 'Обновить описание из источника'}
            </Button>
          )}

          {showPrimaryGroup && <div className="mx-1 h-6 w-px shrink-0 bg-border" aria-hidden="true" />}

          <Button variant="ghost" size="sm" className="h-8 text-muted-foreground hover:text-foreground" onClick={() => setMoveOpen(true)}>
            <Move className="mr-1 size-4" />
            Переместить
          </Button>

          <Button variant="ghost" size="sm" className="h-8 text-muted-foreground hover:text-red-600" onClick={() => setDeleteOpen(true)}>
            <Trash2 className="mr-1 size-4" />
            Удалить
          </Button>

          {translateError && <span className="text-xs text-destructive">{translateError}</span>}
          {refreshError && <span className="text-xs text-destructive">{refreshError}</span>}
          {refreshSuccess && <span className="text-xs text-green-600 dark:text-green-400">Описание обновлено</span>}
          {normalizeError && <span className="text-xs text-destructive">{normalizeError}</span>}
          {normalizeSuccess && <span className="text-xs text-green-600 dark:text-green-400">Нормализация завершена</span>}
          {dumpImagesError && <span className="text-xs text-destructive">{dumpImagesError}</span>}
          {dumpImagesSuccess && <span className="text-xs text-green-600 dark:text-green-400">Выгрузка изображений завершена</span>}
        </div>
      </div>

      <DeleteNodeDialog open={deleteOpen} onOpenChange={setDeleteOpen} node={node} onDeleted={onNavigateHome} />

      <MoveNodeDialog open={moveOpen} onOpenChange={setMoveOpen} node={node} onMoved={handleMoved} />

      {normalizeOpID && (
        <div className="fixed inset-x-0 bottom-0 z-50 m-0 border-t bg-background/95">
          <button
            type="button"
            className="flex w-full items-center justify-between px-4 py-2 text-left text-sm"
            onClick={() => setLogsPanelOpen((v) => !v)}
          >
            <span>Логи нормализации · {normalizing ? `running (${normalizeStage})` : normalizeError ? 'error' : 'success'}</span>
            {logsPanelOpen ? <ChevronDown className="size-4" /> : <ChevronUp className="size-4" />}
          </button>
          <div
            className={cn(
              'overflow-hidden border-t bg-muted/40 transition-all duration-300 ease-out',
              logsPanelOpen ? 'max-h-80 opacity-100' : 'max-h-0 opacity-0',
            )}
            aria-hidden={!logsPanelOpen}
          >
            <div className="px-4 py-2">
              <div className="mb-2 flex items-center justify-between text-xs">
                <span className="text-muted-foreground">Режим логов</span>
                <div className="flex gap-1">
                  <Button
                    type="button"
                    size="sm"
                    variant={logViewMode === 'compact' ? 'default' : 'outline'}
                    className="h-6 px-2 text-xs"
                    onClick={() => setLogViewMode('compact')}
                  >
                    compact
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant={logViewMode === 'raw' ? 'default' : 'outline'}
                    className="h-6 px-2 text-xs"
                    onClick={() => setLogViewMode('raw')}
                  >
                    raw
                  </Button>
                </div>
              </div>
              <div ref={logsContainerRef} className="max-h-56 overflow-auto font-mono text-xs">
              {logEntries.length === 0 ? (
                <div className="text-muted-foreground">Логов пока нет...</div>
              ) : (
                logEntries.map((entry) => (
                  <div key={entry.offset} className="whitespace-pre-wrap break-words py-0.5">
                    <span className="mr-2 text-muted-foreground">[{entry.stream}]</span>
                    {logViewMode === 'compact' ? (
                      <MarkdownContent content={entry.text} nodePath={basePath} />
                    ) : (
                      <span>{entry.text}</span>
                    )}
                  </div>
                ))
              )}
              </div>
            </div>
          </div>
        </div>
      )}

      {dumpImagesOpID && (
        <div className="fixed inset-x-0 bottom-0 z-50 m-0 border-t bg-background/95">
          <button
            type="button"
            className="flex w-full items-center justify-between px-4 py-2 text-left text-sm"
            onClick={() => setDumpLogsPanelOpen((v) => !v)}
          >
            <span>Логи выгрузки изображений · {dumpingImages ? `running (${dumpImagesStage})` : dumpImagesError ? 'error' : 'success'}</span>
            {dumpLogsPanelOpen ? <ChevronDown className="size-4" /> : <ChevronUp className="size-4" />}
          </button>
          <div
            className={cn(
              'overflow-hidden border-t bg-muted/40 transition-all duration-300 ease-out',
              dumpLogsPanelOpen ? 'max-h-80 opacity-100' : 'max-h-0 opacity-0',
            )}
            aria-hidden={!dumpLogsPanelOpen}
          >
            <div className="px-4 py-2">
              <div className="mb-2 flex items-center justify-between text-xs">
                <span className="text-muted-foreground">Режим логов</span>
                <div className="flex gap-1">
                  <Button
                    type="button"
                    size="sm"
                    variant={logViewMode === 'compact' ? 'default' : 'outline'}
                    className="h-6 px-2 text-xs"
                    onClick={() => setLogViewMode('compact')}
                  >
                    compact
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant={logViewMode === 'raw' ? 'default' : 'outline'}
                    className="h-6 px-2 text-xs"
                    onClick={() => setLogViewMode('raw')}
                  >
                    raw
                  </Button>
                </div>
              </div>
              <div ref={dumpLogsContainerRef} className="max-h-56 overflow-auto font-mono text-xs">
                {dumpLogEntries.length === 0 ? (
                  <div className="text-muted-foreground">Логов пока нет...</div>
                ) : (
                  dumpLogEntries.map((entry) => (
                    <div key={entry.offset} className="whitespace-pre-wrap break-words py-0.5">
                      <span className="mr-2 text-muted-foreground">[{entry.stream}]</span>
                      {logViewMode === 'compact' ? (
                        <MarkdownContent content={entry.text} nodePath={basePath} />
                      ) : (
                        <span>{entry.text}</span>
                      )}
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
