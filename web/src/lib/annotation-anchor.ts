import type { NodeAnnotation } from '@/services/api'

/** Plain-text view of markdown for anchor matching (mirrors server resolve). */
export function markdownPlainText(content: string): string {
  const withoutCode = content.replace(/```[\s\S]*?```/g, ' ')
  const lines = withoutCode.split('\n')
  const out: string[] = []
  for (const line of lines) {
    const heading = line.match(/^(#{1,6})\s+(.+)$/)
    if (heading) {
      out.push(stripInlineMarkdown(heading[2]))
      continue
    }
    out.push(stripInlineMarkdown(line))
  }
  return collapseWhitespace(out.join('\n'))
}

function stripInlineMarkdown(text: string): string {
  return text
    .replace(/\*\*(.+?)\*\*/g, '$1')
    .replace(/\*(.+?)\*/g, '$1')
    .replace(/__(.+?)__/g, '$1')
    .replace(/_(.+?)_/g, '$1')
    .replace(/`(.+?)`/g, '$1')
    .replace(/\[(.+?)\]\([^)]+\)/g, '$1')
    .trim()
}

function collapseWhitespace(value: string): string {
  return value.replace(/\s+/g, ' ').trim()
}

export function annotationsBasePath(nodePath: string): string {
  if (!nodePath.includes('.')) return nodePath
  return nodePath.replace(/\.[a-z]{2}$/, '')
}

export function buildAnchorFromSelection(
  contentPath: string,
  content: string,
  exact: string,
  headingId?: string
): NonNullable<NodeAnnotation['anchor']> {
  const plain = markdownPlainText(content)
  const idx = plain.indexOf(exact)
  let prefix = ''
  let suffix = ''
  if (idx >= 0) {
    prefix = plain.slice(Math.max(0, idx - 80), idx)
    suffix = plain.slice(idx + exact.length, idx + exact.length + 80)
  }
  return {
    type: 'text_quote',
    content_path: contentPath,
    exact,
    prefix,
    suffix,
    ...(headingId ? { heading_id: headingId } : {}),
  }
}

export function findHeadingIdForSelection(
  root: HTMLElement,
  range: Range
): string | undefined {
  const headings = root.querySelectorAll('h1[id],h2[id],h3[id],h4[id],h5[id],h6[id]')
  const selectionTop = range.getBoundingClientRect().top
  let lastId: string | undefined
  headings.forEach((heading) => {
    if (heading.getBoundingClientRect().top <= selectionTop + 4) {
      lastId = heading.id
    }
  })
  return lastId
}

export function sortAnnotations(
  notes: NodeAnnotation[],
  content: string
): NodeAnnotation[] {
  const plain = markdownPlainText(content)
  const anchored: { note: NodeAnnotation; pos: number }[] = []
  const general: NodeAnnotation[] = []
  for (const note of notes) {
    if (!note.anchor) {
      general.push(note)
      continue
    }
    const pos = plain.indexOf(note.anchor.exact)
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

export function isResolvedAnnotation(note: NodeAnnotation): boolean {
  return note.resolved === true && !!note.anchor
}
