const envUrl = import.meta.env.VITE_API_URL;
export const API_URL = envUrl === '' ? '' : (envUrl || 'http://localhost:8080');

const fetchOptions: RequestInit = { credentials: 'include' };

async function apiFetch(url: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(url, { ...fetchOptions, ...init })
  if (res.status === 401) {
    const redirect = encodeURIComponent(window.location.pathname + window.location.search)
    window.location.href = `/login?redirect=${redirect}`
    throw new Error('Unauthorized')
  }
  return res
}

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
  manual_processed: boolean;
}

export interface GetNodesParams {
  path?: string;
  recursive?: boolean;
  q?: string;
  type?: string;
  /** When true/false, filters GET /api/nodes by manual_processed. */
  manual_processed?: boolean;
  limit?: number;
  offset?: number;
  sort?: string;
  order?: string;
}

export type WebAuthMode = 'password' | 'google';

export interface SessionStatus {
  authenticated: boolean;
  auth_enabled: boolean;
  /** Present when `auth_enabled` (password vs Google). */
  auth_mode?: WebAuthMode;
}

export async function getSession(): Promise<SessionStatus> {
  const res = await fetch(`${API_URL}/api/auth/session`, fetchOptions);
  if (!res.ok) throw new Error('Failed to get session');
  return res.json();
}

export async function login(login: string, password: string): Promise<void> {
  const res = await fetch(`${API_URL}/api/auth/login`, {
    ...fetchOptions,
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ login, password }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error || 'Login failed');
  }
}

export async function logout(): Promise<void> {
  const res = await fetch(`${API_URL}/api/auth/logout`, {
    ...fetchOptions,
    method: 'POST',
  });
  if (!res.ok) throw new Error('Logout failed');
}

const KB_OAUTH_REDIRECT_KEY = 'kb_oauth_redirect';

/** Start Google OAuth in the same browser (full navigation). */
export function startGoogleOAuth(redirectPath: string): void {
  if (typeof window === 'undefined') return;
  const path = redirectPath.startsWith('/') ? redirectPath : `/${redirectPath}`;
  try {
    window.sessionStorage.setItem(KB_OAUTH_REDIRECT_KEY, path);
  } catch {
    // ignore
  }
  const q = new URLSearchParams();
  if (path !== '/') {
    q.set('redirect', path);
  }
  const suffix = q.toString() ? `?${q.toString()}` : '';
  window.location.assign(`${API_URL}/api/auth/google${suffix}`);
}

export function takeStoredOAuthRedirect(fallback: string): string {
  if (typeof window === 'undefined') return fallback;
  try {
    const s = window.sessionStorage.getItem(KB_OAUTH_REDIRECT_KEY);
    if (s) {
      window.sessionStorage.removeItem(KB_OAUTH_REDIRECT_KEY);
      if (s.startsWith('/') && !s.startsWith('//')) {
        return s;
      }
    }
  } catch {
    // ignore
  }
  return fallback;
}

export async function getTree(): Promise<TreeNode> {
  const res = await apiFetch(`${API_URL}/api/tree`);
  if (!res.ok) throw new Error('Failed to load tree');
  return res.json();
}

export async function getNodes(path: string): Promise<TreeNode[]> {
  const res = await apiFetch(`${API_URL}/api/nodes?path=${encodeURIComponent(path)}`);
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
  if (params.manual_processed === true) {
    searchParams.set('manual_processed', 'true');
  } else if (params.manual_processed === false) {
    searchParams.set('manual_processed', 'false');
  }
  if (params.limit !== undefined) searchParams.set('limit', String(params.limit));
  if (params.offset !== undefined) searchParams.set('offset', String(params.offset));
  if (params.sort) searchParams.set('sort', params.sort);
  if (params.order) searchParams.set('order', params.order);
  const res = await apiFetch(`${API_URL}/api/nodes?${searchParams.toString()}`);
  if (!res.ok) throw new Error('Failed to load nodes');
  const data = await res.json();
  return { nodes: data.nodes || [], total: data.total ?? 0 };
}

export async function getNode(path: string): Promise<Node> {
  const res = await apiFetch(`${API_URL}/api/nodes/${encodeURIComponent(path)}`);
  if (!res.ok) throw new Error('Failed to load node');
  return res.json();
}

export async function patchNodeManualProcessed(
  path: string,
  manual_processed: boolean
): Promise<Node> {
  const encoded = path.split('/').map(encodeURIComponent).join('/');
  const res = await apiFetch(`${API_URL}/api/nodes/${encoded}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ manual_processed }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error || 'Failed to update node');
  }
  return res.json();
}

export interface TranslateStatus {
  status: 'none' | 'pending' | 'in_progress' | 'done' | 'failed';
  error?: string;
}

export async function postTranslate(path: string): Promise<TranslateStatus> {
  const encoded = path.split('/').map(encodeURIComponent).join('/');
  const res = await apiFetch(`${API_URL}/api/articles/translate/${encoded}`, {
    method: 'POST',
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error || 'Translation failed');
  }
  return res.json();
}

export async function getTranslateStatus(path: string): Promise<TranslateStatus> {
  const encoded = path.split('/').map(encodeURIComponent).join('/');
  const res = await apiFetch(`${API_URL}/api/articles/translate/${encoded}`);
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error || 'Failed to get status');
  }
  return res.json();
}

export interface IngestTextOptions {
  typeHint?: 'auto' | 'article' | 'link' | 'note';
  sourceUrl?: string;
  sourceAuthor?: string;
}

export async function ingestText(
  text: string,
  typeHint?: 'auto' | 'article' | 'link' | 'note',
  options?: Pick<IngestTextOptions, 'sourceUrl' | 'sourceAuthor'>
): Promise<Node> {
  const body: Record<string, string> = { text };
  if (typeHint && typeHint !== 'auto') {
    body.type_hint = typeHint;
  }
  if (options?.sourceUrl) {
    body.source_url = options.sourceUrl;
  }
  if (options?.sourceAuthor) {
    body.source_author = options.sourceAuthor;
  }
  const res = await apiFetch(`${API_URL}/api/ingest`, {
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

export interface ImportItem {
  id: number;
  date_unixtime: string;
  text: string;
  source_author: string;
  source_url: string;
}

export interface ImportSessionCreateResponse {
  session_id: string;
  total: number;
  current_index: number;
  current_item: ImportItem | null;
}

export interface ImportSessionState {
  session_id: string;
  total: number;
  current_index: number;
  processed_count: number;
  rejected_count: number;
  current_item: ImportItem | null;
}

export interface ImportAcceptResponse {
  node: Node;
  next_item: ImportItem | null;
}

export interface ImportRejectResponse {
  next_item: ImportItem | null;
}

export async function createImportSession(json: string): Promise<ImportSessionCreateResponse> {
  const res = await apiFetch(`${API_URL}/api/import/telegram`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: json,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error || 'Import failed');
  }
  return res.json();
}

export async function getImportSession(id: string): Promise<ImportSessionState> {
  const res = await apiFetch(`${API_URL}/api/import/telegram/session/${encodeURIComponent(id)}`);
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error || 'Failed to get session');
  }
  return res.json();
}

export async function acceptImportItem(
  id: string,
  typeHint?: 'auto' | 'article' | 'link' | 'note'
): Promise<ImportAcceptResponse> {
  const body = typeHint && typeHint !== 'auto' ? { type_hint: typeHint } : {};
  const res = await apiFetch(`${API_URL}/api/import/telegram/session/${encodeURIComponent(id)}/accept`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error || 'Accept failed');
  }
  return res.json();
}

export async function rejectImportItem(id: string): Promise<ImportRejectResponse> {
  const res = await apiFetch(`${API_URL}/api/import/telegram/session/${encodeURIComponent(id)}/reject`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({}),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error((err as { error?: string }).error || 'Reject failed');
  }
  return res.json();
}
