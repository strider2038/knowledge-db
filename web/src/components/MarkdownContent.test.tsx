/**
 * @vitest-environment jsdom
 */
import type React from 'react'
import { describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { ThemeProvider } from '@/components/theme-provider'
import { TooltipProvider } from '@/components/ui/tooltip'
import { MarkdownContent } from './MarkdownContent'

vi.mock('mermaid', () => ({
  default: {
    initialize: vi.fn(),
    render: vi.fn().mockResolvedValue({
      svg: '<svg data-testid="mermaid-svg">Mermaid Diagram</svg>',
      bindFunctions: undefined,
    }),
  },
}))

function renderWithTheme(ui: React.ReactElement) {
  return render(
    <ThemeProvider defaultTheme="light" attribute="class">
      {ui}
    </ThemeProvider>
  )
}

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
      <TooltipProvider>
        <MarkdownContent
          content={'```javascript\nconst x = 1;\n```'}
        />
      </TooltipProvider>
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

  it('renders mermaid diagrams in code blocks', async () => {
    renderWithTheme(
      <MarkdownContent
        content={
          '```mermaid\ngraph TD\n  A-->B\n```'
        }
      />
    )
    const container = document.querySelector('[data-mermaid-diagram]')
    expect(container).toBeInTheDocument()
    await waitFor(() => {
      const svg = screen.getByTestId('mermaid-svg')
      expect(svg).toBeInTheDocument()
      expect(container).toContainElement(svg)
    })
  })
})
