import { Outlet, Routes, Route } from 'react-router-dom'
import { AuthProvider } from './contexts/AuthContext'
import { GitStatusProvider } from './hooks/useGitStatus'
import { Navbar } from './components/Navbar'
import { ProtectedRoute } from './components/ProtectedRoute'
import { ScrollToTop } from './components/ScrollToTop'
import { AddPage } from './pages/AddPage'
import { LoginPage } from './pages/LoginPage'
import { OverviewPage } from './pages/OverviewPage'
import { NodePage } from './pages/NodePage'

function MainLayout() {
  return (
    <GitStatusProvider>
      <div className="min-h-screen flex flex-col">
        <Navbar />
        <main className="flex-1">
          <Outlet />
        </main>
        <ScrollToTop />
      </div>
    </GitStatusProvider>
  )
}

function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <MainLayout />
            </ProtectedRoute>
          }
        >
          <Route index element={<OverviewPage />} />
          <Route path="add" element={<AddPage />} />
          <Route path="node/*" element={<NodePage />} />
        </Route>
      </Routes>
    </AuthProvider>
  )
}

export default App
