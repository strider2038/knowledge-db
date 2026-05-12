import { useState } from 'react'
import { Check, Languages, Move, RefreshCw, Trash2 } from 'lucide-react'
import { type Node, getTranslateStatus, postTranslate, refreshNodeDescription } from '@/services/api'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { DeleteNodeDialog } from '@/components/DeleteNodeDialog'
import { MoveNodeDialog } from '@/components/MoveNodeDialog'

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
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [moveOpen, setMoveOpen] = useState(false)

  const meta = node.metadata ?? {}
  const nodeType = (meta.type as string) ?? 'note'
  const canTranslate = nodeType === 'article' && !hasTranslations && !isTranslation
  const canRefreshDescription = !isTranslation && typeof meta.source_url === 'string' && meta.source_url.length > 0
  const showManualProcessed = !isTranslation
  const showTranslate = canTranslate

  const handleTranslate = () => {
    setTranslating(true)
    setTranslateError(null)
    postTranslate(basePath)
      .then((status) => {
        if (status.status === 'done') {
          setTranslating(false)
          onNodeChanged(node)
          return
        }
        if (status.status === 'failed') {
          setTranslating(false)
          setTranslateError(status.error ?? 'Ошибка перевода')
          return
        }
        const poll = () => {
          getTranslateStatus(basePath).then((s) => {
            if (s.status === 'done' || s.status === 'failed') {
              setTranslating(false)
              if (s.status === 'failed') setTranslateError(s.error ?? 'Ошибка перевода')
              onNodeChanged(node)
            }
          }).catch(() => {
            setTranslating(false)
            setTranslateError('Ошибка при проверке статуса')
          })
        }
        poll()
        const interval = setInterval(() => {
          getTranslateStatus(basePath).then((s) => {
            if (s.status === 'done' || s.status === 'failed') {
              clearInterval(interval)
              setTranslating(false)
              if (s.status === 'failed') setTranslateError(s.error ?? 'Ошибка перевода')
              onNodeChanged(node)
            }
          }).catch(() => {
            clearInterval(interval)
            setTranslating(false)
            setTranslateError('Ошибка при проверке статуса')
          })
        }, 2500)
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
    refreshNodeDescription(basePath)
      .then((updated) => {
        setRefreshing(false)
        setRefreshSuccess(true)
        onNodeChanged(updated)
      })
      .catch((err) => {
        setRefreshing(false)
        setRefreshError(err instanceof Error ? err.message : 'Не удалось обновить описание')
      })
  }

  const showPrimaryGroup = showManualProcessed || showTranslate || hasTranslations || canRefreshDescription

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
              {manualProcessed ? 'Проверено' : 'Проверено'}
            </Button>
          )}

          {showTranslate && (
            <Button
              variant="outline"
              size="sm"
              className="h-8"
              onClick={handleTranslate}
              disabled={translating}
            >
              <Languages className="mr-1 size-4" />
              {translating ? 'Перевод...' : 'Перевести'}
            </Button>
          )}

          {hasTranslations && (
            <div className="flex flex-wrap gap-1">
              <Button
                variant={currentNodePath === basePath ? 'default' : 'outline'}
                size="sm"
                className="h-8"
                onClick={() => onNavigate(`/node/${basePath}`)}
              >
                Оригинал
              </Button>
              {translations.map((slug) => {
                const transPath = translationPath(basePath, slug)
                const isActive = currentNodePath === transPath
                const langLabel = slug.includes('.') ? slug.split('.').pop() ?? slug : slug
                return (
                  <Button
                    key={slug}
                    variant={isActive ? 'default' : 'outline'}
                    size="sm"
                    className="h-8"
                    onClick={() => onNavigate(`/node/${transPath}`)}
                  >
                    {langLabel}
                  </Button>
                )
              })}
            </div>
          )}

          {canRefreshDescription && (
            <Button
              variant="outline"
              size="sm"
              className="h-8"
              onClick={handleRefreshDescription}
              disabled={refreshing}
            >
              <RefreshCw className={cn('mr-1 size-4', refreshing && 'animate-spin')} />
              {refreshing ? 'Обновление...' : 'Обновить описание из источника'}
            </Button>
          )}

          {showPrimaryGroup && <div className="mx-1 h-6 w-px shrink-0 bg-border" aria-hidden="true" />}

          <Button
            variant="ghost"
            size="sm"
            className="h-8 text-muted-foreground hover:text-foreground"
            onClick={() => setMoveOpen(true)}
          >
            <Move className="mr-1 size-4" />
            Переместить
          </Button>

          <Button
            variant="ghost"
            size="sm"
            className="h-8 text-muted-foreground hover:text-red-600"
            onClick={() => setDeleteOpen(true)}
          >
            <Trash2 className="mr-1 size-4" />
            Удалить
          </Button>

          {translateError && (
            <span className="text-xs text-destructive">{translateError}</span>
          )}
          {refreshError && (
            <span className="text-xs text-destructive">{refreshError}</span>
          )}
          {refreshSuccess && (
            <span className="text-xs text-green-600 dark:text-green-400">Описание обновлено</span>
          )}
        </div>
      </div>

      <DeleteNodeDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        node={node}
        onDeleted={onNavigateHome}
      />

      <MoveNodeDialog
        open={moveOpen}
        onOpenChange={setMoveOpen}
        node={node}
        onMoved={handleMoved}
      />
    </>
  )
}
