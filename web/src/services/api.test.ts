import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { getNodesWithParams } from './api'

describe('getNodesWithParams', () => {
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('builds URL with path and recursive=true', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ nodes: [], total: 0 }),
    })
    await getNodesWithParams({ path: 'programming', recursive: true })
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/api/nodes?'),
      expect.anything()
    )
    const url = new URL(fetchMock.mock.calls[0][0])
    expect(url.searchParams.get('path')).toBe('programming')
    expect(url.searchParams.get('recursive')).toBe('true')
  })

  it('builds URL with q, type, limit, offset, sort, order', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ nodes: [], total: 0 }),
    })
    await getNodesWithParams({
      path: 'ai',
      q: 'go',
      type: 'article,link',
      limit: 20,
      offset: 40,
      sort: 'title',
      order: 'asc',
    })
    const url = new URL(fetchMock.mock.calls[0][0])
    expect(url.searchParams.get('q')).toBe('go')
    expect(url.searchParams.get('type')).toBe('article,link')
    expect(url.searchParams.get('limit')).toBe('20')
    expect(url.searchParams.get('offset')).toBe('40')
    expect(url.searchParams.get('sort')).toBe('title')
    expect(url.searchParams.get('order')).toBe('asc')
  })

  it('returns nodes and total from response', async () => {
    const mockNodes = [
      {
        path: 'topic/node1',
        title: 'Node 1',
        type: 'article',
        created: '2024-01-01T00:00:00Z',
        source_url: 'https://example.com',
        translations: [],
      },
    ]
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ nodes: mockNodes, total: 1 }),
    })
    const result = await getNodesWithParams({ path: 'topic', recursive: true })
    expect(result.nodes).toEqual(mockNodes)
    expect(result.total).toBe(1)
  })

  it('throws when response is not ok', async () => {
    fetchMock.mockResolvedValue({ ok: false })
    await expect(getNodesWithParams({ recursive: true })).rejects.toThrow(
      'Failed to load nodes'
    )
  })
})

describe('ingestText', () => {
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('sends text only when typeHint is auto or omitted', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ path: 'topic/node', annotation: '', content: '', metadata: {} }),
    })
    const { ingestText } = await import('./api')
    await ingestText('my note')
    expect(fetchMock).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ text: 'my note' }),
      })
    )
  })

  it('sends type_hint when typeHint is article', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ path: 'topic/article', annotation: '', content: '', metadata: {} }),
    })
    const { ingestText } = await import('./api')
    await ingestText('https://example.com/article', 'article')
    expect(fetchMock).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ text: 'https://example.com/article', type_hint: 'article' }),
      })
    )
  })

  it('sends type_hint when typeHint is link or note', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ path: 'topic/link', annotation: '', content: '', metadata: {} }),
    })
    const { ingestText } = await import('./api')
    await ingestText('https://pkg.go.dev/net/http', 'link')
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
      text: 'https://pkg.go.dev/net/http',
      type_hint: 'link',
    })
  })

  it('throws with API error message when response not ok', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      json: async () => ({ error: 'ingestion not implemented' }),
    })
    const { ingestText } = await import('./api')
    await expect(ingestText('text')).rejects.toThrow('ingestion not implemented')
  })
})
