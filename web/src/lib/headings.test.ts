import { describe, expect, it } from 'vitest'
import { extractHeadings } from './headings'

describe('extractHeadings', () => {
  it('returns empty array for content without headings', () => {
    expect(extractHeadings('Just a paragraph.')).toEqual([])
    expect(extractHeadings('')).toEqual([])
  })

  it('extracts h1-h6 with level, text, and slug', () => {
    const result = extractHeadings('# Title\n\n## Section One\n\n### Sub')
    expect(result).toHaveLength(3)
    expect(result[0]).toEqual({ level: 1, text: 'Title', slug: 'title' })
    expect(result[1]).toEqual({ level: 2, text: 'Section One', slug: 'section-one' })
    expect(result[2]).toEqual({ level: 3, text: 'Sub', slug: 'sub' })
  })

  it('strips markdown from heading text', () => {
    const result = extractHeadings('## **Bold** and `code` and *italic*')
    expect(result[0].text).toBe('Bold and code and italic')
  })

  it('generates slugs matching github-slugger (lowercase, hyphenated)', () => {
    const result = extractHeadings('# Hello World')
    expect(result[0].slug).toBe('hello-world')
  })

  it('handles duplicate headings with unique slugs', () => {
    const result = extractHeadings('# Same\n\n## Same')
    expect(result[0].slug).toBe('same')
    expect(result[1].slug).toBe('same-1')
  })
})
