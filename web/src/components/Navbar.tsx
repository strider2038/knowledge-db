import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'
import { logout } from '@/services/api'
import { ModeToggle } from './mode-toggle'
import { Button } from './ui/button'

export function Navbar() {
  const location = useLocation()
  const { authenticated, authEnabled, refresh } = useAuth()

  const handleLogout = async () => {
    try {
      await logout()
      await refresh()
      window.location.href = '/login'
    } catch {
      window.location.href = '/login'
    }
  }

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
      <div className="ml-auto flex items-center gap-2">
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
