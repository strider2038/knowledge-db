import { useContext } from 'react'
import { GitStatusContext, type GitStatusContextValue } from '@/contexts/git-status-context'

export function useGitStatus(): GitStatusContextValue {
  const ctx = useContext(GitStatusContext)
  if (!ctx) {
    throw new Error('useGitStatus must be used within GitStatusProvider')
  }
  return ctx
}

export { GitStatusProvider } from '@/contexts/GitStatusProvider'
export type { GitStatusContextValue } from '@/contexts/git-status-context'
