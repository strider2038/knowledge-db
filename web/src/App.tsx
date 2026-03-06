import { Routes, Route } from 'react-router-dom'
import { Navbar } from './components/Navbar'
import { AddPage } from './pages/AddPage'
import { SearchPage } from './pages/SearchPage'
import { NodePage } from './pages/NodePage'

function App() {
  return (
    <div className="min-h-screen flex flex-col">
      <Navbar />
      <main className="flex-1">
        <Routes>
          <Route path="/" element={<SearchPage />} />
          <Route path="/add" element={<AddPage />} />
          <Route path="/node/*" element={<NodePage />} />
        </Routes>
      </main>
    </div>
  )
}

export default App;
