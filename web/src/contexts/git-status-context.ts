import * as React from 'react'
import type { GitStatusResponse } from '@/services/api'

export type GitStatusContextValue = {
  status: GitStatusResponse | null
  loading: boolean
  /** Increments after git sync so open pages reload KB data from the API. */
  dataRevision: number
  refresh: () => Promise<void>
  bumpDataRevision: () => void
}

export const GitStatusContext = React.createContext<GitStatusContextValue | null>(null)
