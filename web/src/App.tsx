import { Routes, Route } from 'react-router-dom';
import { Navbar } from './components/Navbar';
import { AddPage } from './pages/AddPage';
import { SearchPage } from './pages/SearchPage';
import { NodePage } from './pages/NodePage';
import './App.css';

function App() {
  return (
    <>
      <Navbar />
      <Routes>
        <Route path="/" element={<SearchPage />} />
        <Route path="/add" element={<AddPage />} />
        <Route path="/node/*" element={<NodePage />} />
      </Routes>
    </>
  );
}

export default App;
