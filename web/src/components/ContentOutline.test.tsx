/**
 * @vitest-environment jsdom
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { ContentOutline, ContentOutlineFloating } from './ContentOutline'

describe('ContentOutline', () => {
  beforeEach(() => {
    vi.stubGlobal('scrollIntoView', vi.fn())
  })

  it('returns null when content has no headings', () => {
    const { container } = render(<ContentOutline content="Just a paragraph." />)
    expect(container.firstChild).toBeNull()
  })

  it('renders nav with "Содержание" and heading links when content has headings', () => {
    render(
      <ContentOutline
        content={'# Title\n\nSome text.\n\n## Section 1\n\n## Section 2'}
      />
    )
    expect(screen.getByRole('navigation', { name: 'Содержание' })).toBeInTheDocument()
    expect(screen.getByText('Содержание')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Title' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Section 1' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Section 2' })).toBeInTheDocument()
  })

  it('strips markdown from heading text in outline', () => {
    render(<ContentOutline content={'## **Bold** and `code`'} />)
    expect(screen.getByRole('button', { name: 'Bold and code' })).toBeInTheDocument()
  })

  it('scrolls to heading when link is clicked', () => {
    const mockScrollIntoView = vi.fn()
    const el = document.createElement('div')
    el.id = 'section-1'
    el.scrollIntoView = mockScrollIntoView
    document.body.appendChild(el)

    render(<ContentOutline content={'## Section 1'} />)
    fireEvent.click(screen.getByRole('button', { name: 'Section 1' }))

    expect(mockScrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth', block: 'start' })

    document.body.removeChild(el)
  })
})

describe('ContentOutlineFloating', () => {
  beforeEach(() => {
    vi.stubGlobal('scrollIntoView', vi.fn())
  })

  it('returns null when content has no headings', () => {
    const { container } = render(<ContentOutlineFloating content="No headings." />)
    expect(container.firstChild).toBeNull()
  })

  it('renders toggle button with aria-label', () => {
    render(<ContentOutlineFloating content={'# Title'} />)
    expect(screen.getByRole('button', { name: 'Содержание' })).toBeInTheDocument()
  })

  it('shows panel when toggle is clicked', () => {
    render(<ContentOutlineFloating content={'# Title\n## Section'} />)
    const toggle = screen.getByRole('button', { name: 'Содержание' })
    expect(screen.queryByRole('button', { name: 'Title' })).not.toBeInTheDocument()

    fireEvent.click(toggle)
    expect(screen.getByRole('button', { name: 'Title' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Section' })).toBeInTheDocument()
  })

  it('closes panel when heading link is clicked', () => {
    const el = document.createElement('div')
    el.id = 'title'
    el.scrollIntoView = vi.fn()
    document.body.appendChild(el)

    render(<ContentOutlineFloating content={'# Title'} />)
    fireEvent.click(screen.getByRole('button', { name: 'Содержание' }))
    expect(screen.getByRole('button', { name: 'Title' })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Title' }))
    expect(screen.queryByRole('button', { name: 'Title' })).not.toBeInTheDocument()

    document.body.removeChild(el)
  })
})
