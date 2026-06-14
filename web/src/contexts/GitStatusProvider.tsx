import * as React from 'react'
import { getGitStatus, type GitStatusResponse } from '@/services/api'
import { GitStatusContext } from './git-status-context'

const POLL_INTERVAL = 30_000

export function GitStatusProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = React.useState<GitStatusResponse | null>(null)
  const [loading, setLoading] = React.useState(false)
  const [dataRevision, setDataRevision] = React.useState(0)
  const intervalRef = React.useRef<ReturnType<typeof setInterval> | null>(null)

  const bumpDataRevision = React.useCallback(() => {
    setDataRevision((n) => n + 1)
  }, [])

  const refresh = React.useCallback(async () => {
    try {
      const s = await getGitStatus()
      setStatus(s)
    } catch {
      setStatus(null)
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    void refresh()
    intervalRef.current = setInterval(() => void refresh(), POLL_INTERVAL)
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [refresh])

  const value = React.useMemo(
    () => ({ status, loading, dataRevision, refresh, bumpDataRevision }),
    [status, loading, dataRevision, refresh, bumpDataRevision],
  )

  return <GitStatusContext.Provider value={value}>{children}</GitStatusContext.Provider>
}
