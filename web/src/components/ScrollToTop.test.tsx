/**
 * @vitest-environment jsdom
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { ScrollToTop } from './ScrollToTop'

const SCROLL_THRESHOLD = 300

function setScrollY(value: number) {
  Object.defineProperty(window, 'scrollY', {
    value,
    writable: true,
    configurable: true,
  })
}

describe('ScrollToTop', () => {
  beforeEach(() => {
    vi.stubGlobal('scrollTo', vi.fn())
    setScrollY(0)
  })

  it('returns null when scroll position is below threshold', () => {
    setScrollY(SCROLL_THRESHOLD - 1)
    const { container } = render(<ScrollToTop />)
    expect(container.firstChild).toBeNull()
  })

  it('shows button when scroll position exceeds threshold', () => {
    setScrollY(SCROLL_THRESHOLD + 1)
    render(<ScrollToTop />)
    expect(screen.getByRole('button', { name: 'Прокрутить вверх' })).toBeInTheDocument()
  })

  it('calls window.scrollTo when button is clicked', () => {
    const mockScrollTo = vi.fn()
    vi.stubGlobal('scrollTo', mockScrollTo)
    setScrollY(SCROLL_THRESHOLD + 1)

    render(<ScrollToTop />)
    fireEvent.click(screen.getByRole('button', { name: 'Прокрутить вверх' }))

    expect(mockScrollTo).toHaveBeenCalledWith({ top: 0, behavior: 'smooth' })
  })
})
