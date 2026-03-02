import { useState } from 'react';
import { ingestText } from '../services/api';

export function AddPage() {
  const [text, setText] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!text.trim()) return;
    setLoading(true);
    setError(null);
    setSuccess(false);
    try {
      await ingestText(text.trim());
      setSuccess(true);
      setText('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ padding: '1rem', maxWidth: 600 }}>
      <h2>Добавить</h2>
      <form onSubmit={handleSubmit}>
        <textarea
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Введите текст..."
          rows={8}
          style={{ width: '100%', marginBottom: '0.5rem' }}
          disabled={loading}
        />
        <button type="submit" disabled={loading || !text.trim()}>
          {loading ? 'Отправка...' : 'Добавить'}
        </button>
      </form>
      {error && <p style={{ color: 'red' }}>{error}</p>}
      {success && <p style={{ color: 'green' }}>Добавлено</p>}
    </div>
  );
}
