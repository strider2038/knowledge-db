import { Link, useLocation } from 'react-router-dom';

export function Navbar() {
  const location = useLocation();
  return (
    <nav style={{ padding: '0.5rem 1rem', borderBottom: '1px solid #ccc', display: 'flex', gap: '1rem' }}>
      <Link to="/" style={{ fontWeight: location.pathname === '/' ? 'bold' : 'normal' }}>Поиск</Link>
      <Link to="/add" style={{ fontWeight: location.pathname === '/add' ? 'bold' : 'normal' }}>Добавить</Link>
    </nav>
  );
}
