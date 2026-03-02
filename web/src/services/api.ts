const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export interface TreeNode {
  name: string;
  path: string;
  children?: TreeNode[];
}

export interface Node {
  path: string;
  annotation: string;
  content: string;
  metadata: Record<string, unknown>;
}

export async function getTree(): Promise<TreeNode> {
  const res = await fetch(`${API_URL}/api/tree`);
  if (!res.ok) throw new Error('Failed to load tree');
  return res.json();
}

export async function getNodes(path: string): Promise<TreeNode[]> {
  const res = await fetch(`${API_URL}/api/nodes?path=${encodeURIComponent(path)}`);
  if (!res.ok) throw new Error('Failed to load nodes');
  const data = await res.json();
  return data.nodes || [];
}

export async function getNode(path: string): Promise<Node> {
  const res = await fetch(`${API_URL}/api/nodes/${encodeURIComponent(path)}`);
  if (!res.ok) throw new Error('Failed to load node');
  return res.json();
}

export async function ingestText(text: string): Promise<Node> {
  const res = await fetch(`${API_URL}/api/ingest`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error || 'Ingestion failed');
  }
  return res.json();
}
