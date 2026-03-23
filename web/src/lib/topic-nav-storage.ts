import type { TopicNavPersisted } from './topic-nav-types'

export const TOPIC_NAV_STORAGE_KEY = 'kb.overview.topicNav.v1' as const

export const defaultTopicNavPersisted = (): TopicNavPersisted => ({
  defaultExpandDepth: 2,
  userExpanded: [],
  userCollapsed: [],
})

export function loadTopicNavPersisted(): TopicNavPersisted {
  try {
    const raw = localStorage.getItem(TOPIC_NAV_STORAGE_KEY)
    if (!raw) return defaultTopicNavPersisted()
    const parsed = JSON.parse(raw) as Partial<TopicNavPersisted>
    const base = defaultTopicNavPersisted()
    const depth =
      typeof parsed.defaultExpandDepth === 'number' &&
      Number.isFinite(parsed.defaultExpandDepth)
        ? Math.min(8, Math.max(1, Math.floor(parsed.defaultExpandDepth)))
        : base.defaultExpandDepth
    return {
      defaultExpandDepth: depth,
      userExpanded: Array.isArray(parsed.userExpanded)
        ? parsed.userExpanded.filter((x): x is string => typeof x === 'string')
        : [],
      userCollapsed: Array.isArray(parsed.userCollapsed)
        ? parsed.userCollapsed.filter((x): x is string => typeof x === 'string')
        : [],
    }
  } catch {
    return defaultTopicNavPersisted()
  }
}

export function saveTopicNavPersisted(next: TopicNavPersisted): void {
  try {
    localStorage.setItem(TOPIC_NAV_STORAGE_KEY, JSON.stringify(next))
  } catch {
    // ignore quota / private mode
  }
}
