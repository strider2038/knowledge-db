import * as React from 'react'

/** Plain text from markdown inline/block children. */
export function flattenMarkdownText(node: React.ReactNode): string {
  if (node == null || typeof node === 'boolean') return ''
  if (typeof node === 'string' || typeof node === 'number') return String(node)
  if (Array.isArray(node)) return node.map(flattenMarkdownText).join('')
  if (React.isValidElement<{ children?: React.ReactNode }>(node)) {
    return flattenMarkdownText(node.props.children)
  }
  return ''
}
