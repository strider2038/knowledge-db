import { Link, useLocation } from 'react-router-dom'
import { ModeToggle } from './mode-toggle'

export function Navbar() {
  const location = useLocation()
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
      <div className="ml-auto">
        <ModeToggle />
      </div>
    </nav>
  )
}
