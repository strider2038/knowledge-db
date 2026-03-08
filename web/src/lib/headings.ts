import GitHubSlugger from 'github-slugger'

export interface Heading {
  level: number
  text: string
  slug: string
}

/**
 * Strip inline markdown formatting for display.
 */
function stripMarkdown(text: string): string {
  return text
    .replace(/\*\*(.+?)\*\*/g, '$1')
    .replace(/\*(.+?)\*/g, '$1')
    .replace(/__(.+?)__/g, '$1')
    .replace(/_(.+?)_/g, '$1')
    .replace(/`(.+?)`/g, '$1')
    .replace(/\[(.+?)\]\([^)]+\)/g, '$1')
    .trim()
}

/**
 * Extract headings (h1-h6) from markdown content.
 * Uses github-slugger for slugs to match rehype-slug output.
 */
export function extractHeadings(markdown: string): Heading[] {
  const slugger = new GitHubSlugger()
  const lines = markdown.split('\n')
  const headings: Heading[] = []

  for (const line of lines) {
    const match = line.match(/^(#{1,6})\s+(.+)$/)
    if (match) {
      const level = match[1].length
      const rawText = match[2].trim()
      const slug = slugger.slug(rawText)
      const text = stripMarkdown(rawText)
      headings.push({ level, text, slug })
    }
  }

  return headings
}
