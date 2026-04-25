import * as React from 'react'
import type { GitStatusResponse } from '@/services/api'

export type GitStatusContextValue = {
  status: GitStatusResponse | null
  loading: boolean
  refresh: () => Promise<void>
}

export const GitStatusContext = React.createContext<GitStatusContextValue | null>(null)
