import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from './errors'
import { clearCsrfToken, request, setUnauthorizedListener } from './http'

const json = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

describe('request', () => {
  const fetchMock = vi.fn<typeof fetch>()

  beforeEach(() => {
    clearCsrfToken()
    fetchMock.mockReset()
    vi.stubGlobal('fetch', fetchMock)
  })
  afterEach(() => {
    setUnauthorizedListener(null)
    vi.unstubAllGlobals()
  })

  it('fetches the CSRF token once and sends it on protected mutations', async () => {
    fetchMock
      .mockResolvedValueOnce(json(200, { csrf_token: 'tok-1' }))
      .mockResolvedValueOnce(json(201, { id: 'b1' }))
      .mockResolvedValueOnce(json(201, { id: 'b2' }))

    await request('/v1/question-banks', { method: 'POST', json: { title: 'A' }, csrf: true })
    await request('/v1/question-banks', { method: 'POST', json: { title: 'B' }, csrf: true })

    expect(fetchMock).toHaveBeenCalledTimes(3)
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/v1/auth/csrf')
    for (const call of fetchMock.mock.calls.slice(1)) {
      const init = call[1]!
      expect(new Headers(init.headers).get('X-CSRF-Token')).toBe('tok-1')
      expect(init.credentials).toBe('same-origin')
    }
  })

  it('refreshes a rejected CSRF token once and re-sends the request', async () => {
    fetchMock
      .mockResolvedValueOnce(json(200, { csrf_token: 'old' }))
      .mockResolvedValueOnce(json(403, { error: 'invalid or missing CSRF token' }))
      .mockResolvedValueOnce(json(200, { csrf_token: 'new' }))
      .mockResolvedValueOnce(json(200, { ok: true }))

    const { data } = await request<{ ok: boolean }>('/v1/attempts/a/finish', { method: 'POST', csrf: true })

    expect(data.ok).toBe(true)
    expect(new Headers(fetchMock.mock.calls[3]?.[1]?.headers).get('X-CSRF-Token')).toBe('new')
  })

  it('never retries a network failure on its own', async () => {
    fetchMock
      .mockResolvedValueOnce(json(200, { csrf_token: 'tok' }))
      .mockRejectedValueOnce(new TypeError('Failed to fetch'))

    const error = await request('/v1/question-banks/b/imports', { method: 'POST', json: {}, csrf: true }).catch(
      (e: unknown) => e,
    )

    expect(error).toBeInstanceOf(ApiError)
    expect((error as ApiError).kind).toBe('network')
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('resolves 204 without reading a body', async () => {
    fetchMock
      .mockResolvedValueOnce(json(200, { csrf_token: 'tok' }))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))

    const response = await request('/v1/quizzes/q', { method: 'DELETE', csrf: true })
    expect(response).toEqual({ status: 204, data: undefined })
  })

  it('notifies the app about 401 on protected routes, but not for a failed login', async () => {
    const listener = vi.fn()
    setUnauthorizedListener(listener)
    fetchMock
      .mockResolvedValueOnce(json(401, { error: 'unauthorized' }))
      .mockResolvedValueOnce(json(401, { error: 'invalid email or password' }))

    await expect(request('/v1/question-banks')).rejects.toMatchObject({ kind: 'unauthorized' })
    await expect(
      request('/v1/auth/login', { method: 'POST', json: {}, context: 'login' }),
    ).rejects.toMatchObject({ kind: 'invalid_credentials' })
    expect(listener).toHaveBeenCalledTimes(1)
  })

  it('maps 403 "email verification required" and keeps Retry-After', async () => {
    fetchMock
      .mockResolvedValueOnce(json(403, { error: 'email verification required' }))
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ error: 'too many requests' }), { status: 429, headers: { 'Retry-After': '30' } }),
      )

    await expect(request('/v1/auth/login', { method: 'POST', json: {}, context: 'login' })).rejects.toMatchObject({
      kind: 'email_unverified',
    })
    await expect(request('/v1/auth/resend-verification', { method: 'POST', json: {} })).rejects.toMatchObject({
      kind: 'rate_limited',
      retryAfter: 30,
    })
  })
})
