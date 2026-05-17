import { Link, useLocation } from 'react-router-dom'
import { GitCommit, LogOut, Menu, RefreshCw } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { logout, postGitCommit, postGitSync, getIndexStatus } from '@/services/api'
import { useGitStatus } from '@/hooks/useGitStatus'
import { toast } from '@/hooks/use-toast'
import { ModeToggle } from './mode-toggle'
import { Button } from './ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { useState, useEffect } from 'react'
import { cn } from '@/lib/utils'

const GIT_DISABLED_HINT =
  'На сервере отключён git (например KB_GIT_DISABLED=true). Сохранение в репозиторий через интерфейс недоступно — используйте git вручную в каталоге базы.'

function navLinkClass(active: boolean) {
  return cn(
    active ? 'font-semibold text-foreground' : 'text-muted-foreground hover:text-foreground',
  )
}

export function Navbar() {
  const location = useLocation()
  const { authenticated, authEnabled, refresh } = useAuth()
  const { status: gitStatus, refresh: refreshGit } = useGitStatus()
  const [committing, setCommitting] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [chatAvailable, setChatAvailable] = useState(false)
  const [searchAvailable, setSearchAvailable] = useState(false)
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  useEffect(() => {
    getIndexStatus()
      .then((status) => {
        setChatAvailable(true)
        setSearchAvailable(status.keyword_index === 'fts5' || status.keyword_index === 'scan')
      })
      .catch(() => {
        setChatAvailable(false)
        setSearchAvailable(false)
      })
  }, [])

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

  const handleRefresh = async () => {
    setSyncing(true)
    try {
      const result = await postGitSync()
      toast({
        title: 'Обновлено',
        description: result.message,
      })
      await refreshGit()
    } catch (err) {
      toast({
        variant: 'destructive',
        title: 'Не удалось обновить',
        description: err instanceof Error ? err.message : 'Ошибка запроса',
      })
    } finally {
      setSyncing(false)
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

  const p = location.pathname

  const mobileNav = (
    <nav className="mt-4 flex flex-col gap-1">
      <SheetClose asChild>
        <Link
          to="/"
          className={cn('rounded-md px-3 py-2.5 text-base hover:bg-accent', navLinkClass(p === '/'))}
        >
          Обзор
        </Link>
      </SheetClose>
      <SheetClose asChild>
        <Link
          to="/add"
          className={cn('rounded-md px-3 py-2.5 text-base hover:bg-accent', navLinkClass(p === '/add'))}
        >
          Добавить
        </Link>
      </SheetClose>
      {searchAvailable && (
        <SheetClose asChild>
          <Link
            to="/search"
            className={cn(
              'rounded-md px-3 py-2.5 text-base hover:bg-accent',
              navLinkClass(p === '/search'),
            )}
          >
            Поиск
          </Link>
        </SheetClose>
      )}
      {chatAvailable && (
        <SheetClose asChild>
          <Link
            to="/chat"
            className={cn(
              'rounded-md px-3 py-2.5 text-base hover:bg-accent',
              navLinkClass(p === '/chat'),
            )}
          >
            Чат
          </Link>
        </SheetClose>
      )}
    </nav>
  )

  return (
    <nav className="flex h-14 min-h-14 items-center gap-2 border-b px-2 sm:gap-3 sm:px-4">
      <Sheet open={mobileMenuOpen} onOpenChange={setMobileMenuOpen}>
        <SheetTrigger asChild>
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="shrink-0 md:hidden"
            aria-label="Открыть меню разделов"
          >
            <Menu className="size-5" />
          </Button>
        </SheetTrigger>
        <SheetContent side="left" className="w-[min(100%,280px)] p-0 sm:max-w-[280px]">
          <SheetHeader className="border-b p-4 text-left">
            <SheetTitle className="text-base">Разделы</SheetTitle>
          </SheetHeader>
          <div className="p-2">{mobileNav}</div>
        </SheetContent>
      </Sheet>

      <div className="hidden min-w-0 flex-1 items-center gap-4 md:flex">
        <Link to="/" className={navLinkClass(p === '/')}>
          Обзор
        </Link>
        <Link to="/add" className={navLinkClass(p === '/add')}>
          Добавить
        </Link>
        {searchAvailable && (
          <Link to="/search" className={navLinkClass(p === '/search')}>
            Поиск
          </Link>
        )}
        {chatAvailable && (
          <Link to="/chat" className={navLinkClass(p === '/chat')}>
            Чат
          </Link>
        )}
      </div>

      <div className="ml-auto flex min-w-0 shrink-0 items-center gap-1 sm:gap-2">
        {showSaveArea && (
          <div className="flex items-center gap-1">
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="inline-flex max-w-full">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="h-9 gap-0 px-2 sm:gap-1.5 sm:px-3"
                    disabled={gitDisabled || syncing || committing}
                    onClick={() => void handleRefresh()}
                    aria-label={syncing ? 'Обновление...' : 'Обновить'}
                  >
                    <RefreshCw
                      className={cn('size-4 shrink-0 sm:mr-1.5', syncing && 'animate-spin')}
                      aria-hidden
                    />
                    <span className="hidden sm:inline">{syncing ? 'Обновление...' : 'Обновить'}</span>
                  </Button>
                </span>
              </TooltipTrigger>
              <TooltipContent side="bottom" className="max-w-xs text-balance">
                {gitDisabled
                  ? GIT_DISABLED_HINT
                  : 'Подтянуть изменения с удалённого репозитория (git fetch и merge с origin/main).'}
              </TooltipContent>
            </Tooltip>
            {!saveActive && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="inline-flex max-w-full">
                    <Button type="button" variant="outline" size="sm" className="h-9 px-2 sm:px-3" disabled>
                      <GitCommit className="size-4 sm:mr-1.5 sm:hidden" aria-hidden />
                      <span className="hidden sm:inline">{saveLabelIdle}</span>
                    </Button>
                  </span>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="max-w-xs text-balance">
                  {gitDisabled ? GIT_DISABLED_HINT : 'Нет незакоммиченных изменений в репозитории базы. Кнопка станет активной после правок файлов.'}
                </TooltipContent>
              </Tooltip>
            )}
            {saveActive && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="h-9 gap-0 px-2 sm:gap-1.5 sm:px-3"
                    disabled={committing || syncing}
                    onClick={() => void handleSave()}
                    aria-label={saveLabelActive}
                  >
                    <GitCommit className="size-4 shrink-0 sm:mr-1.5" aria-hidden />
                    <span className="hidden sm:inline">{saveLabelActive}</span>
                    <span className="pl-1 text-xs tabular-nums sm:hidden" aria-hidden>
                      {gitStatus?.changed_files ?? 0}
                    </span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="max-w-xs sm:hidden">
                  {saveLabelActive}
                </TooltipContent>
              </Tooltip>
            )}
          </div>
        )}
        {authEnabled && authenticated && (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="h-9 px-2 sm:px-3"
                onClick={handleLogout}
                aria-label="Выход"
              >
                <LogOut className="size-4 sm:mr-1.5" />
                <span className="hidden sm:inline">Выход</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom" className="sm:hidden">
              Выход
            </TooltipContent>
          </Tooltip>
        )}
        <ModeToggle />
      </div>
    </nav>
  )
}
