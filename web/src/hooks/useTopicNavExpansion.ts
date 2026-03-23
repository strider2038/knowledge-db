import { useCallback, useMemo, useState } from 'react'
import {
  isForcedOpenPath,
  isTopicBranchExpanded,
} from '@/lib/topic-nav-expansion'
import { loadTopicNavPersisted, saveTopicNavPersisted } from '@/lib/topic-nav-storage'
import type { TopicNavPersisted } from '@/lib/topic-nav-types'

export function useTopicNavExpansion(currentPath: string) {
  const [state, setState] = useState<TopicNavPersisted>(() =>
    loadTopicNavPersisted()
  )

  const userExpandedSet = useMemo(
    () => new Set(state.userExpanded),
    [state.userExpanded]
  )
  const userCollapsedSet = useMemo(
    () => new Set(state.userCollapsed),
    [state.userCollapsed]
  )

  const setDefaultExpandDepth = useCallback((depth: number) => {
    const d = Math.min(8, Math.max(1, Math.floor(depth)))
    setState((prev) => {
      const next = { ...prev, defaultExpandDepth: d }
      saveTopicNavPersisted(next)
      return next
    })
  }, [])

  const isExpanded = useCallback(
    (path: string, hasChildren: boolean) =>
      isTopicBranchExpanded(
        path,
        hasChildren,
        currentPath,
        state.defaultExpandDepth,
        userExpandedSet,
        userCollapsedSet
      ),
    [
      currentPath,
      state.defaultExpandDepth,
      userCollapsedSet,
      userExpandedSet,
    ]
  )

  const toggleBranch = useCallback(
    (path: string, hasChildren: boolean) => {
      if (!hasChildren || isForcedOpenPath(path, currentPath)) return
      setState((prev) => {
        const expSet = new Set(prev.userExpanded)
        const colSet = new Set(prev.userCollapsed)
        const expanded = isTopicBranchExpanded(
          path,
          true,
          currentPath,
          prev.defaultExpandDepth,
          expSet,
          colSet
        )
        let userExpanded = [...prev.userExpanded]
        let userCollapsed = [...prev.userCollapsed]
        if (expanded) {
          if (!userCollapsed.includes(path)) userCollapsed = [...userCollapsed, path]
          userExpanded = userExpanded.filter((p) => p !== path)
        } else {
          userCollapsed = userCollapsed.filter((p) => p !== path)
          if (!userExpanded.includes(path)) userExpanded = [...userExpanded, path]
        }
        const next: TopicNavPersisted = {
          ...prev,
          userExpanded,
          userCollapsed,
        }
        saveTopicNavPersisted(next)
        return next
      })
    },
    [currentPath]
  )

  return {
    defaultExpandDepth: state.defaultExpandDepth,
    setDefaultExpandDepth,
    isExpanded,
    toggleBranch,
  }
}
