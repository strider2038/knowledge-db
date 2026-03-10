export const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

/** URL для статики базы (изображения, вложения узлов). */
export function getAssetUrl(path: string): string {
  const encoded = path.split('/').map(encodeURIComponent).join('/');
  return `${API_URL}/api/assets/${encoded}`;
}

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

export interface NodeListItem {
  path: string;
  title: string;
  type: string;
  created: string;
  source_url: string;
  translations?: string[];
  annotation?: string;
  keywords?: string[];
}

export interface GetNodesParams {
  path?: string;
  recursive?: boolean;
  q?: string;
  type?: string;
  limit?: number;
  offset?: number;
  sort?: string;
  order?: string;
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

export async function getNodesWithParams(
  params: GetNodesParams
): Promise<{ nodes: NodeListItem[]; total: number }> {
  const searchParams = new URLSearchParams();
  if (params.path !== undefined && params.path !== '') searchParams.set('path', params.path);
  searchParams.set('recursive', 'true');
  if (params.q) searchParams.set('q', params.q);
  if (params.type) searchParams.set('type', params.type);
  if (params.limit !== undefined) searchParams.set('limit', String(params.limit));
  if (params.offset !== undefined) searchParams.set('offset', String(params.offset));
  if (params.sort) searchParams.set('sort', params.sort);
  if (params.order) searchParams.set('order', params.order);
  const res = await fetch(`${API_URL}/api/nodes?${searchParams.toString()}`);
  if (!res.ok) throw new Error('Failed to load nodes');
  const data = await res.json();
  return { nodes: data.nodes || [], total: data.total ?? 0 };
}

export async function getNode(path: string): Promise<Node> {
  const res = await fetch(`${API_URL}/api/nodes/${encodeURIComponent(path)}`);
  if (!res.ok) throw new Error('Failed to load node');
  return res.json();
}

export async function ingestText(
  text: string,
  typeHint?: 'auto' | 'article' | 'link' | 'note'
): Promise<Node> {
  const body =
    typeHint && typeHint !== 'auto'
      ? { text, type_hint: typeHint }
      : { text };
  const res = await fetch(`${API_URL}/api/ingest`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error || 'Ingestion failed');
  }
  return res.json();
}
