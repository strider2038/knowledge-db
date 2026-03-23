/** Число сегментов пути темы (например `a/b` → 2). */
export function pathSegmentDepth(path: string): number {
  if (!path) return 0
  return path.split('/').filter(Boolean).length
}

/** Пути всех предков, которые должны быть раскрыты, чтобы `path` был виден. */
export function ancestorPathPrefixes(path: string): string[] {
  if (!path) return []
  const parts = path.split('/').filter(Boolean)
  if (parts.length <= 1) return []
  const out: string[] = []
  let acc = ''
  for (let i = 0; i < parts.length - 1; i++) {
    acc = i === 0 ? parts[0]! : `${acc}/${parts[i]}`
    out.push(acc)
  }
  return out
}

export function isForcedOpenPath(path: string, currentPath: string): boolean {
  return ancestorPathPrefixes(currentPath).includes(path)
}

export function isTopicBranchExpanded(
  path: string,
  hasChildren: boolean,
  currentPath: string,
  defaultExpandDepth: number,
  userExpanded: ReadonlySet<string>,
  userCollapsed: ReadonlySet<string>
): boolean {
  if (!hasChildren) return false
  if (isForcedOpenPath(path, currentPath)) return true
  if (userCollapsed.has(path)) return false
  return (
    pathSegmentDepth(path) <= defaultExpandDepth || userExpanded.has(path)
  )
}
