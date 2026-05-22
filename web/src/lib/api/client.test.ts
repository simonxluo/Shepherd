import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiClient } from './client'

describe('ApiClient', () => {
  beforeEach(() => {
    global.fetch = vi.fn()
  })

  it('GET fetch calls correct URL and returns parsed JSON', async () => {
    const fakeJson = { ok: true }
    // @ts-expect-error mock global fetch
    global.fetch.mockResolvedValue({ ok: true, status: 200, json: async () => fakeJson })
    const client = new ApiClient('https://api.example.com')
    const data = await client.get('/test')
    expect(data).toEqual(fakeJson)
    // @ts-expect-error mock global fetch
    expect(global.fetch.mock.calls[0][0]).toContain('/test')
    // @ts-expect-error mock global fetch
    expect(global.fetch.mock.calls[0][1]).toMatchObject({ method: 'GET' })
  })

  it('POST fetch sends JSON body and returns parsed JSON', async () => {
    const fakeJson = { created: true }
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => fakeJson })
    global.fetch = mockFetch
    const client = new ApiClient('https://api.example.com')
    const body = { name: 'test' }
    const data = await client.post('/items', body)
    expect(data).toEqual(fakeJson)
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining('items'),
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
    )
  })
})
