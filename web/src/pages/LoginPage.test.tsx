/**
 * @vitest-environment jsdom
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider, useAuth } from '@/contexts/AuthContext'
import { LoginPage } from './LoginPage'
import { Navbar } from '@/components/Navbar'

const mockLogin = vi.fn()
const mockGetSession = vi.fn()
const mockLogout = vi.fn()

vi.mock('@/services/api', () => ({
  getSession: (...args: unknown[]) => mockGetSession(...args),
  login: (...args: unknown[]) => mockLogin(...args),
  logout: (...args: unknown[]) => mockLogout(...args),
  takeStoredOAuthRedirect: (fallback: string) => fallback,
  startGoogleOAuth: vi.fn(),
}))

function renderLogin(initialEntry = '/login') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/" element={<div>Overview</div>} />
          <Route path="/add" element={<div>Add Page</div>} />
          <Route path="*" element={<Navigate to="/login" replace />} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>
  )
}

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetSession.mockResolvedValue({ authenticated: false, auth_enabled: true, auth_mode: 'password' })
  })

  it('renders login form', async () => {
    renderLogin()
    await waitFor(() => {
      expect(screen.getByLabelText('Логин')).toBeTruthy()
      expect(screen.getByLabelText('Пароль')).toBeTruthy()
      expect(screen.getByRole('button', { name: 'Войти' })).toBeTruthy()
    })
  })

  it('shows error on failed login', async () => {
    mockLogin.mockRejectedValue(new Error('invalid credentials'))
    renderLogin()

    const loginInput = await screen.findByLabelText('Логин')
    const passwordInput = screen.getByLabelText('Пароль')
    const submitBtn = screen.getByRole('button', { name: 'Войти' })

    fireEvent.change(loginInput, { target: { value: 'user' } })
    fireEvent.change(passwordInput, { target: { value: 'wrong' } })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(screen.getByText(/invalid credentials|Неверный логин или пароль/)).toBeTruthy()
    })
  })

  it('redirects to overview on successful login', async () => {
    mockLogin.mockResolvedValue(undefined)
    mockGetSession
      .mockResolvedValueOnce({ authenticated: false, auth_enabled: true, auth_mode: 'password' })
      .mockResolvedValueOnce({ authenticated: true, auth_enabled: true, auth_mode: 'password' })
    renderLogin()

    const loginInput = await screen.findByLabelText('Логин')
    const passwordInput = screen.getByLabelText('Пароль')
    const submitBtn = screen.getByRole('button', { name: 'Войти' })

    fireEvent.change(loginInput, { target: { value: 'user' } })
    fireEvent.change(passwordInput, { target: { value: 'pass' } })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(screen.getByText('Overview')).toBeTruthy()
    })
  })

  it('redirects to redirect param after successful login', async () => {
    mockLogin.mockResolvedValue(undefined)
    mockGetSession
      .mockResolvedValueOnce({ authenticated: false, auth_enabled: true, auth_mode: 'password' })
      .mockResolvedValueOnce({ authenticated: true, auth_enabled: true, auth_mode: 'password' })
    renderLogin('/login?redirect=/add')

    const loginInput = await screen.findByLabelText('Логин')
    const passwordInput = screen.getByLabelText('Пароль')
    const submitBtn = screen.getByRole('button', { name: 'Войти' })

    fireEvent.change(loginInput, { target: { value: 'user' } })
    fireEvent.change(passwordInput, { target: { value: 'pass' } })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(screen.getByText('Add Page')).toBeTruthy()
    })
  })
})

describe('ProtectedRoute', () => {
  function ProtectedRoute({ children }: { children: React.ReactNode }) {
    const { authenticated, loading } = useAuth()
    if (loading) return <div>Loading...</div>
    if (!authenticated) return <Navigate to="/login" replace />
    return <>{children}</>
  }

  it('redirects unauthenticated user to login', async () => {
    mockGetSession.mockResolvedValue({ authenticated: false, auth_enabled: true, auth_mode: 'password' })
    render(
      <MemoryRouter initialEntries={['/']}>
        <AuthProvider>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/" element={<ProtectedRoute><div>Protected</div></ProtectedRoute>} />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByLabelText('Логин')).toBeTruthy()
    })
  })

  it('shows protected content when authenticated', async () => {
    mockGetSession.mockResolvedValue({ authenticated: true, auth_enabled: true, auth_mode: 'password' })
    render(
      <MemoryRouter initialEntries={['/']}>
        <AuthProvider>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/" element={<ProtectedRoute><div>Protected Content</div></ProtectedRoute>} />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText('Protected Content')).toBeTruthy()
    })
  })
})

describe('Logout', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockLogout.mockResolvedValue(undefined)
  })

  it('calls logout and redirects to login', async () => {
    mockGetSession.mockResolvedValue({ authenticated: true, auth_enabled: true, auth_mode: 'password' })
    const locationSpy = { href: '' }
    Object.defineProperty(window, 'location', {
      value: { ...window.location, get href() { return locationSpy.href }, set href(v: string) { locationSpy.href = v } },
      configurable: true,
    })

    render(
      <MemoryRouter initialEntries={['/']}>
        <AuthProvider>
          <Navbar />
        </AuthProvider>
      </MemoryRouter>
    )

    const logoutBtn = await screen.findByRole('button', { name: 'Выход' })
    fireEvent.click(logoutBtn)

    await waitFor(() => {
      expect(mockLogout).toHaveBeenCalledTimes(1)
    })
    await waitFor(() => {
      expect(locationSpy.href).toBe('/login')
    })
  })
})
