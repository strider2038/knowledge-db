import type { NodeAnnotation } from '@/services/api'

export function annotationsBasePath(nodePath: string): string {
  if (!nodePath.includes('.')) return nodePath
  return nodePath.replace(/\.[a-z]{2}$/, '')
}

export function isResolvedInContent(
  content: string,
  anchor: NonNullable<NodeAnnotation['anchor']>
): boolean {
  if (!anchor.exact) return false
  const candidates = [content, collapseWhitespace(content)]
  return candidates.some((text) => matchAnchorInText(text, anchor))
}

function matchAnchorInText(
  text: string,
  anchor: NonNullable<NodeAnnotation['anchor']>
): boolean {
  const exact = anchor.exact
  if (!text.includes(exact)) return false
  if (!anchor.prefix && !anchor.suffix) return true
  let start = 0
  while (start < text.length) {
    const idx = text.indexOf(exact, start)
    if (idx < 0) return false
    if (contextMatches(text, idx, exact.length, anchor.prefix ?? '', anchor.suffix ?? '')) {
      return true
    }
    start = idx + 1
  }
  return false
}

function contextMatches(
  text: string,
  pos: number,
  exactLen: number,
  prefix: string,
  suffix: string
): boolean {
  if (prefix) {
    const beforeStart = pos - prefix.length
    if (beforeStart < 0 || text.slice(beforeStart, pos) !== prefix) return false
  }
  if (suffix) {
    const afterStart = pos + exactLen
    if (afterStart + suffix.length > text.length) return false
    if (text.slice(afterStart, afterStart + suffix.length) !== suffix) return false
  }
  return true
}

function collapseWhitespace(value: string): string {
  return value.replace(/\s+/g, ' ').trim()
}

export function buildAnchorFromSelection(
  contentPath: string,
  content: string,
  exact: string
): NonNullable<NodeAnnotation['anchor']> {
  const idx = content.indexOf(exact)
  let prefix = ''
  let suffix = ''
  if (idx >= 0) {
    prefix = content.slice(Math.max(0, idx - 80), idx)
    suffix = content.slice(idx + exact.length, idx + exact.length + 80)
  }
  return {
    type: 'text_quote',
    content_path: contentPath,
    exact,
    prefix,
    suffix,
  }
}

export function sortAnnotations(
  notes: NodeAnnotation[],
  content: string
): NodeAnnotation[] {
  const anchored: { note: NodeAnnotation; pos: number }[] = []
  const general: NodeAnnotation[] = []
  for (const note of notes) {
    if (!note.anchor) {
      general.push(note)
      continue
    }
    const pos = content.indexOf(note.anchor.exact)
    anchored.push({ note, pos: pos >= 0 ? pos : Number.MAX_SAFE_INTEGER })
  }
  anchored.sort((a, b) => a.pos - b.pos)
  general.sort(
    (a, b) => new Date(b.updated).getTime() - new Date(a.updated).getTime()
  )
  return [...anchored.map((item) => item.note), ...general]
}

export function scrollToTextQuote(
  container: HTMLElement | null,
  exact: string
): boolean {
  if (!container || !exact) return false
  const walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT)
  let node: Text | null
  while ((node = walker.nextNode() as Text | null)) {
    const idx = node.textContent?.indexOf(exact) ?? -1
    if (idx >= 0) {
      const range = document.createRange()
      range.setStart(node, idx)
      range.setEnd(node, idx + exact.length)
      const el =
        node.parentElement?.closest('p, li, blockquote, h1, h2, h3, h4, td') ??
        node.parentElement
      el?.scrollIntoView({ behavior: 'smooth', block: 'center' })
      return true
    }
  }
  return false
}
