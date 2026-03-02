import { useEffect, useState } from 'react';
import { useLocation, Link } from 'react-router-dom';
import { getNode, type Node } from '../services/api';

export function NodePage() {
  const location = useLocation();
  const path = location.pathname.replace(/^\/node\/?/, '');
  const [node, setNode] = useState<Node | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!path) return;
    getNode(path)
      .then(setNode)
      .catch((err) => setError(err instanceof Error ? err.message : 'Ошибка'))
      .finally(() => setLoading(false));
  }, [path]);

  if (loading) return <p>Загрузка...</p>;
  if (error) return <p style={{ color: 'red' }}>{error}</p>;
  if (!node) return null;

  return (
    <div style={{ padding: '1rem', maxWidth: 800 }}>
      <p><Link to="/">← Назад</Link></p>
      <h2>{node.path}</h2>
      <section style={{ marginBottom: '1rem' }}>
        <h3>Аннотация</h3>
        <pre style={{ whiteSpace: 'pre-wrap' }}>{node.annotation || '(нет)'}</pre>
      </section>
      <section style={{ marginBottom: '1rem' }}>
        <h3>Содержание</h3>
        <pre style={{ whiteSpace: 'pre-wrap' }}>{node.content || '(нет)'}</pre>
      </section>
      <section>
        <h3>Метаданные</h3>
        <pre>{JSON.stringify(node.metadata, null, 2)}</pre>
      </section>
    </div>
  );
}
