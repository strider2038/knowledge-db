import { describe, expect, it } from 'vitest'
import {
  ancestorPathPrefixes,
  isForcedOpenPath,
  isTopicBranchExpanded,
  pathSegmentDepth,
} from './topic-nav-expansion'

describe('topic-nav-expansion', () => {
  it('pathSegmentDepth', () => {
    expect(pathSegmentDepth('')).toBe(0)
    expect(pathSegmentDepth('a')).toBe(1)
    expect(pathSegmentDepth('a/b')).toBe(2)
  })

  it('ancestorPathPrefixes', () => {
    expect(ancestorPathPrefixes('')).toEqual([])
    expect(ancestorPathPrefixes('a')).toEqual([])
    expect(ancestorPathPrefixes('a/b')).toEqual(['a'])
    expect(ancestorPathPrefixes('a/b/c')).toEqual(['a', 'a/b'])
  })

  it('isForcedOpenPath', () => {
    expect(isForcedOpenPath('a', 'a/b/c')).toBe(true)
    expect(isForcedOpenPath('a/b', 'a/b/c')).toBe(true)
    expect(isForcedOpenPath('a/b/c', 'a/b/c')).toBe(false)
    expect(isForcedOpenPath('x', 'a/b')).toBe(false)
  })

  it('isTopicBranchExpanded: must open ancestors', () => {
    expect(
      isTopicBranchExpanded('a', true, 'a/b', 1, new Set(), new Set())
    ).toBe(true)
    expect(
      isTopicBranchExpanded('x/y', true, 'a/b', 1, new Set(), new Set())
    ).toBe(false)
  })

  it('isTopicBranchExpanded: default depth (N = visible levels)', () => {
    expect(
      isTopicBranchExpanded('a', true, '', 1, new Set(), new Set())
    ).toBe(false)
    expect(
      isTopicBranchExpanded('a', true, '', 2, new Set(), new Set())
    ).toBe(true)
    expect(
      isTopicBranchExpanded('a/b', true, '', 2, new Set(), new Set())
    ).toBe(false)
    expect(
      isTopicBranchExpanded('a/b', true, '', 3, new Set(), new Set())
    ).toBe(true)
    expect(
      isTopicBranchExpanded('a/b/c', true, '', 3, new Set(), new Set())
    ).toBe(false)
  })

  it('isTopicBranchExpanded: userCollapsed', () => {
    expect(
      isTopicBranchExpanded('a', true, '', 2, new Set(), new Set(['a']))
    ).toBe(false)
  })

  it('isTopicBranchExpanded: userExpanded', () => {
    expect(
      isTopicBranchExpanded(
        'a/b/c',
        true,
        '',
        1,
        new Set(['a/b/c']),
        new Set()
      )
    ).toBe(true)
  })
})
