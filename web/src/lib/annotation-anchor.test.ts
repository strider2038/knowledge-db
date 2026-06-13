import { describe, expect, it } from 'vitest'
import {
  annotationsBasePath,
  buildAnchorFromSelection,
  isResolvedAnnotation,
  markdownPlainText,
} from './annotation-anchor'
import type { NodeAnnotation } from '@/services/api'

describe('annotation-anchor', () => {
  it('strips inline markdown for plain text', () => {
    expect(markdownPlainText('**bold** and `code`')).toBe('bold and code')
  })

  it('builds anchor prefix/suffix from plain text', () => {
    const anchor = buildAnchorFromSelection('topic/node', 'Intro **exact** tail', 'exact')
    expect(anchor.exact).toBe('exact')
    expect(anchor.content_path).toBe('topic/node')
    expect(anchor.prefix).toContain('Intro')
    expect(anchor.suffix).toContain('tail')
  })

  it('normalizes translation path to base', () => {
    expect(annotationsBasePath('topic/node.ru')).toBe('topic/node')
  })

  it('trusts server resolved flag for markers', () => {
    const resolved: NodeAnnotation = {
      id: '1',
      created: '',
      updated: '',
      body: 'note',
      anchor: {
        type: 'text_quote',
        content_path: 'topic/node',
        exact: 'text',
      },
      resolved: true,
    }
    const unresolved: NodeAnnotation = { ...resolved, resolved: false }
    expect(isResolvedAnnotation(resolved)).toBe(true)
    expect(isResolvedAnnotation(unresolved)).toBe(false)
  })
})
