/**
 * @vitest-environment jsdom
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MarkdownContent } from './MarkdownContent'

describe('MarkdownContent', () => {
  it('renders markdown headings, lists, and paragraphs', () => {
    render(
      <MarkdownContent
        content={
          '# Title\n\nA paragraph with **bold**.\n\n- Item 1\n- Item 2'
        }
      />
    )
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Title')
    expect(screen.getByText(/A paragraph with/)).toBeInTheDocument()
    expect(screen.getByText('Item 1')).toBeInTheDocument()
    expect(screen.getByText('Item 2')).toBeInTheDocument()
  })

  it('renders GFM tables', () => {
    render(
      <MarkdownContent
        content={
          '| A | B |\n| --- | --- |\n| 1 | 2 |'
        }
      />
    )
    expect(screen.getByRole('table')).toBeInTheDocument()
    expect(screen.getByText('A')).toBeInTheDocument()
    expect(screen.getByText('1')).toBeInTheDocument()
  })

  it('renders code blocks with highlight.js classes', () => {
    render(
      <MarkdownContent
        content={'```javascript\nconst x = 1;\n```'}
      />
    )
    const code = document.querySelector('pre code')
    expect(code).toBeInTheDocument()
    expect(code?.className).toMatch(/hljs|language-javascript/)
  })

  it('renders links with target="_blank" and rel="noopener noreferrer"', () => {
    render(
      <MarkdownContent content={'[Example](https://example.com)'} />
    )
    const link = screen.getByRole('link', { name: 'Example' })
    expect(link).toHaveAttribute('href', 'https://example.com')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })
})
