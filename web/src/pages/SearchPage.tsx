import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getTree, getNodes, type TreeNode } from '../services/api';

export function SearchPage() {
  const [tree, setTree] = useState<TreeNode | null>(null);
  const [nodes, setNodes] = useState<TreeNode[]>([]);
  const [selectedPath, setSelectedPath] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getTree()
      .then(setTree)
      .catch((err) => setError(err instanceof Error ? err.message : 'Ошибка'))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!selectedPath) {
      setNodes([]);
      return;
    }
    getNodes(selectedPath)
      .then(setNodes)
      .catch(() => setNodes([]));
  }, [selectedPath]);

  const renderTree = (node: TreeNode, depth = 0) => {
    if (!node.children?.length) return null;
    return (
      <ul key={node.path || 'root'} style={{ marginLeft: depth * 16, listStyle: 'none' }}>
        {node.children.map((child) => (
          <li key={child.path}>
            <button
              type="button"
              onClick={() => setSelectedPath(child.path)}
              style={{
                background: selectedPath === child.path ? '#e0e0e0' : 'transparent',
                border: 'none',
                cursor: 'pointer',
                textAlign: 'left',
                padding: 4,
              }}
            >
              {child.name}
            </button>
            {renderTree(child, depth + 1)}
          </li>
        ))}
      </ul>
    );
  };

  if (loading) return <p>Загрузка...</p>;
  if (error) return <p style={{ color: 'red' }}>{error}</p>;

  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 60px)' }}>
      <aside style={{ width: 250, borderRight: '1px solid #ccc', padding: '1rem', overflow: 'auto' }}>
        <h3>Темы</h3>
        {tree && renderTree(tree)}
      </aside>
      <main style={{ flex: 1, padding: '1rem', overflow: 'auto' }}>
        <h3>Узлы</h3>
        {nodes.length === 0 ? (
          <p>Выберите тему слева</p>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={{ textAlign: 'left', padding: 8 }}>Название</th>
                <th style={{ textAlign: 'left', padding: 8 }}>Путь</th>
              </tr>
            </thead>
            <tbody>
              {nodes.map((n) => (
                <tr key={n.path}>
                  <td style={{ padding: 8 }}>
                    <Link to={`/node/${n.path}`}>{n.name}</Link>
                  </td>
                  <td style={{ padding: 8 }}>{n.path}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </main>
    </div>
  );
}
