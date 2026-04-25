import { useState } from 'react'
import { Navigate, useSearchParams } from 'react-router-dom'
import { login, startGoogleOAuth, takeStoredOAuthRedirect } from '@/services/api'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

const OAUTH_ERR: Record<string, string> = {
  forbidden: 'Доступ запрещён. Ваш адрес не в списке разрешённых.',
  state: 'Сессия входа устарела. Попробуйте снова.',
  oauth: 'Ошибка авторизации Google. Попробуйте снова.',
  server: 'Ошибка сервера при входе. Попробуйте позже.',
  config: 'Сервер настроен неверно.',
}

function oauthErrorMessage(code: string | null): string | null {
  if (!code) return null
  return OAUTH_ERR[code] ?? 'Не удалось войти. Попробуйте снова.'
}

export function LoginPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const rawRedirect = searchParams.get('redirect') || '/'
  const { authenticated, authEnabled, authMode, loading: authLoading, refresh } = useAuth()
  const [loginValue, setLoginValue] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(() => oauthErrorMessage(searchParams.get('error')))
  const [loading, setLoading] = useState(false)
  const isGoogle = authMode === 'google' && authEnabled

  if (authLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    )
  }

  if (authenticated) {
    const dest = takeStoredOAuthRedirect(rawRedirect)
    return <Navigate to={dest} replace />
  }

  const clearOAuthErrorInUrl = () => {
    if (!searchParams.get('error')) return
    const next = new URLSearchParams(searchParams)
    next.delete('error')
    setSearchParams(next, { replace: true })
  }

  const handleGoogleClick = () => {
    setError(null)
    clearOAuthErrorInUrl()
    startGoogleOAuth(rawRedirect)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!loginValue.trim() || !password) return
    setLoading(true)
    setError(null)
    try {
      await login(loginValue.trim(), password)
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Неверный логин или пароль')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Вход</CardTitle>
        </CardHeader>
        <CardContent>
          {isGoogle && (
            <div className="mb-4">
              <Button
                type="button"
                variant="outline"
                className="w-full"
                onClick={handleGoogleClick}
                disabled={loading}
              >
                Войти через Google
              </Button>
            </div>
          )}
          {!isGoogle && (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="login" className="text-sm font-medium">
                  Логин
                </label>
                <input
                  id="login"
                  type="text"
                  value={loginValue}
                  onChange={(e) => setLoginValue(e.target.value)}
                  autoComplete="username"
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                  placeholder="Логин"
                  disabled={loading}
                />
              </div>
              <div className="space-y-2">
                <label htmlFor="password" className="text-sm font-medium">
                  Пароль
                </label>
                <input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                  placeholder="Пароль"
                  disabled={loading}
                />
              </div>
              {error && (
                <p className="text-sm text-destructive">{error}</p>
              )}
              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? 'Вход...' : 'Войти'}
              </Button>
            </form>
          )}
          {isGoogle && error && (
            <p className="text-sm text-destructive mt-2">{error}</p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
