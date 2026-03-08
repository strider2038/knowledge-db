import { Routes, Route } from 'react-router-dom'
import { Navbar } from './components/Navbar'
import { ScrollToTop } from './components/ScrollToTop'
import { AddPage } from './pages/AddPage'
import { OverviewPage } from './pages/OverviewPage'
import { NodePage } from './pages/NodePage'

function App() {
  return (
    <div className="min-h-screen flex flex-col">
      <Navbar />
      <main className="flex-1">
        <Routes>
          <Route path="/" element={<OverviewPage />} />
          <Route path="/add" element={<AddPage />} />
          <Route path="/node/*" element={<NodePage />} />
        </Routes>
      </main>
      <ScrollToTop />
    </div>
  )
}

export default App;
