import { ApiError, classify, type RequestContext } from './errors'

type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export interface RequestOptions {
  method?: Method
  json?: unknown
  form?: FormData
  query?: Record<string, string | number | undefined>
  signal?: AbortSignal
  /** Protected mutations need X-CSRF-Token. Auth bootstrap calls (login, register…) don't. */
  csrf?: boolean
  context?: RequestContext
}

export interface RawResponse<T> {
  status: number
  data: T
}

// ── Interface language: emails from register/resend follow it ──────────────
let language = 'ru'

/** Called by the i18n provider; sent as Accept-Language on every request. */
export function setRequestLanguage(lang: string) {
  language = lang
}

// ── CSRF token: memory only, bound to the current session ──────────────────
let csrfToken: string | null = null
let csrfInFlight: Promise<string> | null = null

export function clearCsrfToken() {
  csrfToken = null
  csrfInFlight = null
}

async function getCsrfToken(): Promise<string> {
  if (csrfToken) return csrfToken
  csrfInFlight ??= send<{ csrf_token: string }>('/v1/auth/csrf', {})
    .then(({ data }) => {
      csrfToken = data.csrf_token
      return data.csrf_token
    })
    .finally(() => {
      csrfInFlight = null
    })
  return csrfInFlight
}

// ── 401 hook: the app clears session state and redirects to login ──────────
type UnauthorizedListener = () => void
let onUnauthorized: UnauthorizedListener | null = null

export function setUnauthorizedListener(listener: UnauthorizedListener | null) {
  onUnauthorized = listener
}

function buildUrl(path: string, query?: RequestOptions['query']) {
  if (!query) return path
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined) params.set(key, String(value))
  }
  const qs = params.toString()
  return qs ? `${path}?${qs}` : path
}

async function send<T>(path: string, options: RequestOptions, token?: string): Promise<RawResponse<T>> {
  const headers = new Headers({ Accept: 'application/json', 'Accept-Language': language })
  let body: BodyInit | undefined
  if (options.json !== undefined) {
    headers.set('Content-Type', 'application/json')
    body = JSON.stringify(options.json)
  } else if (options.form) {
    // Let the browser set multipart boundary.
    body = options.form
  }
  if (token) headers.set('X-CSRF-Token', token)

  let response: Response
  try {
    response = await fetch(buildUrl(path, options.query), {
      method: options.method ?? 'GET',
      headers,
      body,
      credentials: 'same-origin',
      signal: options.signal,
    })
  } catch (cause) {
    if (cause instanceof DOMException && cause.name === 'AbortError') throw cause
    throw new ApiError({ kind: 'network', status: 0, cause })
  }

  if (response.status === 204) {
    return { status: 204, data: undefined as T }
  }

  const text = await response.text()
  let payload: unknown = undefined
  if (text) {
    try {
      payload = JSON.parse(text)
    } catch {
      payload = undefined
    }
  }

  if (!response.ok) {
    const serverMessage =
      payload && typeof payload === 'object' && 'error' in payload && typeof payload.error === 'string'
        ? payload.error
        : undefined
    const retryAfterHeader = response.headers.get('Retry-After')
    const retryAfter = retryAfterHeader ? Number.parseInt(retryAfterHeader, 10) : undefined
    const kind = classify(response.status, serverMessage, options.context ?? 'default')
    throw new ApiError({
      kind,
      status: response.status,
      serverMessage,
      retryAfter: Number.isFinite(retryAfter) ? retryAfter : undefined,
    })
  }

  return { status: response.status, data: payload as T }
}

/**
 * Single entry point for backend calls. Relative `/v1/...` URLs on the page
 * origin; the session is an HttpOnly cookie; CSRF token travels in a header.
 *
 * A rejected CSRF token (403 before the handler runs) is refreshed once and
 * the request is re-sent — safe even for creations because the server did not
 * process the first one. Network failures are never retried here.
 */
export async function request<T>(path: string, options: RequestOptions = {}): Promise<RawResponse<T>> {
  try {
    const token = options.csrf ? await getCsrfToken() : undefined
    try {
      return await send<T>(path, options, token)
    } catch (error) {
      if (options.csrf && error instanceof ApiError && error.kind === 'csrf') {
        clearCsrfToken()
        return await send<T>(path, options, await getCsrfToken())
      }
      throw error
    }
  } catch (error) {
    if (error instanceof ApiError && error.kind === 'unauthorized') {
      clearCsrfToken()
      onUnauthorized?.()
    }
    throw error
  }
}
