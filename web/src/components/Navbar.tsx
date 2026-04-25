import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'
import { logout, postGitCommit } from '@/services/api'
import { useGitStatus } from '@/hooks/useGitStatus'
import { toast } from '@/hooks/use-toast'
import { ModeToggle } from './mode-toggle'
import { Button } from './ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { useState } from 'react'

const GIT_DISABLED_HINT =
  'На сервере отключён git (например KB_GIT_DISABLED=true). Сохранение в репозиторий через интерфейс недоступно — используйте git вручную в каталоге базы.'

export function Navbar() {
  const location = useLocation()
  const { authenticated, authEnabled, refresh } = useAuth()
  const { status: gitStatus, refresh: refreshGit } = useGitStatus()
  const [committing, setCommitting] = useState(false)

  const handleLogout = async () => {
    try {
      await logout()
      await refresh()
      window.location.href = '/login'
    } catch {
      window.location.href = '/login'
    }
  }

  const handleSave = async () => {
    setCommitting(true)
    try {
      const result = await postGitCommit()
      if (result.committed) {
        toast({
          title: 'Сохранено',
          description: result.message,
        })
        await refreshGit()
      } else {
        toast({
          title: 'Нет изменений',
          description: result.message,
        })
      }
    } catch (err) {
      toast({
        variant: 'destructive',
        title: 'Не удалось сохранить',
        description: err instanceof Error ? err.message : 'Ошибка запроса',
      })
    } finally {
      setCommitting(false)
    }
  }

  const gitKnown = gitStatus !== null
  const gitDisabled = gitStatus?.git_disabled === true
  const hasLocalChanges = gitStatus?.has_changes === true
  const showSaveArea = gitKnown
  const saveActive = !gitDisabled && hasLocalChanges

  const saveLabelActive = committing
    ? 'Сохранение...'
    : `Сохранить (${gitStatus?.changed_files ?? 0})`

  const saveLabelIdle = gitDisabled ? '⚠️ Сохранить' : 'Сохранить'

  return (
    <nav className="flex h-14 items-center gap-4 border-b px-4">
      <Link
        to="/"
        className={
          location.pathname === '/'
            ? 'font-semibold text-foreground'
            : 'text-muted-foreground hover:text-foreground'
        }
      >
        Обзор
      </Link>
      <Link
        to="/add"
        className={
          location.pathname === '/add'
            ? 'font-semibold text-foreground'
            : 'text-muted-foreground hover:text-foreground'
        }
      >
        Добавить
      </Link>
      <div className="ml-auto flex min-w-0 flex-wrap items-center justify-end gap-2">
        {showSaveArea && !saveActive && (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="inline-flex max-w-full">
                <Button type="button" variant="outline" size="sm" disabled>
                  {saveLabelIdle}
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent side="bottom" className="max-w-xs text-balance">
              {gitDisabled ? GIT_DISABLED_HINT : 'Нет незакоммиченных изменений в репозитории базы. Кнопка станет активной после правок файлов.'}
            </TooltipContent>
          </Tooltip>
        )}
        {showSaveArea && saveActive && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={committing}
            onClick={() => void handleSave()}
          >
            {saveLabelActive}
          </Button>
        )}
        {authEnabled && authenticated && (
          <Button variant="ghost" size="sm" onClick={handleLogout}>
            Выход
          </Button>
        )}
        <ModeToggle />
      </div>
    </nav>
  )
}
