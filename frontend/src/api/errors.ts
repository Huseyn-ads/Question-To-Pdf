export type ApiErrorKind =
  | 'network'
  | 'unauthorized' // 401 on a protected route: no valid session
  | 'invalid_credentials' // 401 from POST /auth/login
  | 'email_unverified' // 403 "email verification required" from login
  | 'csrf' // 403 "invalid or missing CSRF token"
  | 'forbidden' // other 403 (origin, Google profile)
  | 'not_found'
  | 'validation' // 400
  | 'conflict' // 409 (attempt closed, email taken, …)
  | 'too_large' // 413
  | 'unsupported' // 415
  | 'unprocessable' // 422 (broken / unsupported DOCX)
  | 'rate_limited' // 429
  | 'unavailable' // 503
  | 'server' // 5xx, unexpected

export class ApiError extends Error {
  readonly kind: ApiErrorKind
  readonly status: number
  /** Backend `{ "error": "..." }` text (English, for diagnostics and mapping). */
  readonly serverMessage: string | undefined
  /** Seconds from Retry-After, when present. */
  readonly retryAfter: number | undefined

  constructor(init: {
    kind: ApiErrorKind
    status: number
    serverMessage?: string
    retryAfter?: number
    cause?: unknown
  }) {
    super(init.serverMessage ?? init.kind, { cause: init.cause })
    this.name = 'ApiError'
    this.kind = init.kind
    this.status = init.status
    this.serverMessage = init.serverMessage
    this.retryAfter = init.retryAfter
  }
}

export function isApiError(error: unknown, kind?: ApiErrorKind): error is ApiError {
  return error instanceof ApiError && (kind === undefined || error.kind === kind)
}

export type RequestContext = 'default' | 'login'

export function classify(status: number, message: string | undefined, context: RequestContext): ApiErrorKind {
  const text = (message ?? '').toLowerCase()
  switch (status) {
    case 400:
      return 'validation'
    case 401:
      return context === 'login' ? 'invalid_credentials' : 'unauthorized'
    case 403:
      if (text.includes('email verification required')) return 'email_unverified'
      if (text.includes('csrf')) return 'csrf'
      return 'forbidden'
    case 404:
      return 'not_found'
    case 409:
      return 'conflict'
    case 413:
      return 'too_large'
    case 415:
      return 'unsupported'
    case 422:
      return 'unprocessable'
    case 429:
      return 'rate_limited'
    case 503:
      return 'unavailable'
    default:
      return 'server'
  }
}

/** Errors worth retrying automatically for safe (GET / idempotent) requests. */
export function isTransient(error: unknown): boolean {
  return isApiError(error) && (error.kind === 'network' || error.kind === 'server' || error.kind === 'unavailable')
}
