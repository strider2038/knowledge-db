/**
 * @vitest-environment jsdom
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { AddPage } from './AddPage'

const { ingestText } = vi.hoisted(() => ({
  ingestText: vi.fn(),
}))

vi.mock('../services/api', () => ({
  ingestText,
}))

function renderAddPage() {
  return render(
    <MemoryRouter initialEntries={['/add']}>
      <Routes>
        <Route path="/add" element={<AddPage />} />
      </Routes>
    </MemoryRouter>
  )
}

describe('AddPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders form with type selector, textarea and submit button', () => {
    renderAddPage()
    expect(screen.getByRole('button', { name: 'Авто' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Статья' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Ссылка' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Заметка' })).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Введите текст...')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Добавить' })).toBeInTheDocument()
  })

  it('submit button is disabled when text is empty', () => {
    renderAddPage()
    expect(screen.getByRole('button', { name: 'Добавить' })).toBeDisabled()
  })

  it('submit button is enabled when text is entered', () => {
    renderAddPage()
    fireEvent.change(screen.getByPlaceholderText('Введите текст...'), {
      target: { value: 'some text' },
    })
    expect(screen.getByRole('button', { name: 'Добавить' })).toBeEnabled()
  })

  it('shows hint when article or link is selected', () => {
    renderAddPage()
    expect(screen.queryByText('Вставьте URL в текст')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Статья' }))
    expect(screen.getByText('Вставьте URL в текст')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Ссылка' }))
    expect(screen.getByText('Вставьте URL в текст')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Заметка' }))
    expect(screen.queryByText('Вставьте URL в текст')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Авто' }))
    expect(screen.queryByText('Вставьте URL в текст')).not.toBeInTheDocument()
  })

  it('calls ingestText with text and typeHint on submit', async () => {
    ingestText.mockResolvedValue({ path: 'topic/new-node' })
    renderAddPage()

    fireEvent.change(screen.getByPlaceholderText('Введите текст...'), {
      target: { value: 'my note' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Добавить' }))

    expect(ingestText).toHaveBeenCalledWith('my note', 'auto')
  })

  it('calls ingestText with type_hint when article selected', async () => {
    ingestText.mockResolvedValue({ path: 'topic/article' })
    renderAddPage()

    fireEvent.click(screen.getByRole('button', { name: 'Статья' }))
    fireEvent.change(screen.getByPlaceholderText('Введите текст...'), {
      target: { value: 'https://example.com/article' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Добавить' }))

    expect(ingestText).toHaveBeenCalledWith('https://example.com/article', 'article')
  })

  it('shows success with link to node on successful submit', async () => {
    ingestText.mockResolvedValue({ path: 'go/concurrency/new-note' })
    renderAddPage()

    fireEvent.change(screen.getByPlaceholderText('Введите текст...'), {
      target: { value: 'notes about goroutines' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Добавить' }))

    expect(await screen.findByText(/Добавлено/)).toBeInTheDocument()
    const link = screen.getByRole('link', { name: 'Перейти к узлу' })
    expect(link).toHaveAttribute('href', '/node/go/concurrency/new-note')
  })

  it('shows error message on failed submit', async () => {
    ingestText.mockRejectedValue(new Error('LLM unavailable'))
    renderAddPage()

    fireEvent.change(screen.getByPlaceholderText('Введите текст...'), {
      target: { value: 'some text' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Добавить' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('LLM unavailable')
  })

  it('shows loading state during submit', async () => {
    let resolveIngest: (value: { path: string }) => void
    ingestText.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveIngest = resolve
        })
    )
    renderAddPage()

    fireEvent.change(screen.getByPlaceholderText('Введите текст...'), {
      target: { value: 'text' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Добавить' }))

    expect(screen.getByRole('button', { name: /Обработка/ })).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Введите текст...')).toBeDisabled()

    resolveIngest!({ path: 'done' })
    await screen.findByText(/Добавлено/)
  })
})
