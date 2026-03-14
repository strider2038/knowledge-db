/* eslint-disable react-refresh/only-export-components -- useAuth is a hook, not a component */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'
import { getSession, type SessionStatus } from '@/services/api'

type AuthContextValue = {
  authenticated: boolean | null
  authEnabled: boolean | null
  loading: boolean
  refresh: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<SessionStatus | null>(null)
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    try {
      const s = await getSession()
      setSession(s)
    } catch {
      setSession({ authenticated: false, auth_enabled: true })
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  const value: AuthContextValue = {
    authenticated: session?.authenticated ?? null,
    authEnabled: session?.auth_enabled ?? null,
    loading,
    refresh,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return ctx
}
