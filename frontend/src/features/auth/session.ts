import { useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { useApi } from '@/api/context'
import { isApiError } from '@/api/errors'
import { clearCsrfToken } from '@/api/http'
import { queryKeys } from '@/api/queryKeys'
import type { User } from '@/api/types'

/**
 * Current user: `User` for a live session, `null` when the backend answers
 * 401. Every screen reads the profile through this hook so the session has
 * one source of truth in the query cache.
 */
export function useMe() {
  const api = useApi()
  const queryClient = useQueryClient()
  return useQuery<User | null>({
    queryKey: queryKeys.me,
    queryFn: async ({ signal }) => {
      try {
        const user = await api.auth.me(signal)
        // Another account signed in (e.g. in a different tab): drop every
        // cached resource of the previous user before showing anything.
        const previous = queryClient.getQueryData<User | null>(queryKeys.me)
        if (previous && previous.id !== user.id) dropUserData(queryClient)
        return user
      } catch (error) {
        if (isApiError(error, 'unauthorized')) return null
        throw error
      }
    },
    staleTime: 5 * 60_000,
  })
}

/** Removes everything cached for the signed-in user except the profile entry. */
export function dropUserData(queryClient: QueryClient) {
  queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== queryKeys.me[0] })
}

/** Start a new session view: forget the old user's data, then store the new profile. */
export function beginSession(queryClient: QueryClient, user: User) {
  clearCsrfToken()
  queryClient.clear()
  queryClient.setQueryData(queryKeys.me, user)
}

export type SessionEndReason = 'logout' | 'expired'

/**
 * End the session locally (after logout or a 401). The reason lives next to
 * the profile in the cache so the route guard can choose where to send the
 * user without racing a separate navigation.
 */
export function endSession(queryClient: QueryClient, reason?: SessionEndReason) {
  clearCsrfToken()
  queryClient.setQueryData(queryKeys.sessionEnd, reason ?? null)
  queryClient.setQueryData(queryKeys.me, null)
  dropUserData(queryClient)
}

export function getSessionEndReason(queryClient: QueryClient): SessionEndReason | null {
  return queryClient.getQueryData<SessionEndReason | null>(queryKeys.sessionEnd) ?? null
}

// ── Safe return URLs ────────────────────────────────────────────────────────
const AUTH_PATHS = ['/login', '/register', '/check-email', '/auth/']

/** Where a signed-in user lands by default. `/` is the public landing page. */
export const APP_HOME = '/dashboard'

/**
 * Accepts only same-origin app paths. Rejects absolute and protocol-relative
 * URLs (`//evil`, `/\evil`), backend routes and the auth pages themselves.
 */
export function safeNext(raw: string | null | undefined, fallback = APP_HOME): string {
  if (!raw || raw.length > 2048) return fallback
  if (!raw.startsWith('/') || raw.startsWith('//') || raw.startsWith('/\\')) return fallback
  if ([...raw].some((char) => char.charCodeAt(0) < 0x20)) return fallback
  let url: URL
  try {
    url = new URL(raw, 'http://app.invalid')
  } catch {
    return fallback
  }
  if (url.origin !== 'http://app.invalid') return fallback
  const path = url.pathname
  // After signing in, the landing page is not a destination: open the app.
  if (path === '/') return fallback
  if (path.startsWith('/v1/') || path === '/v1' || path === '/health' || path === '/ready') return fallback
  if (AUTH_PATHS.some((prefix) => path === prefix || path.startsWith(prefix.endsWith('/') ? prefix : `${prefix}/`)))
    return fallback
  return `${path}${url.search}${url.hash}`
}

export function loginPath(next?: string) {
  const target = next ? safeNext(next, '') : ''
  return target && target !== APP_HOME ? `/login?next=${encodeURIComponent(target)}` : '/login'
}

// Google sign-in leaves the SPA, so the return path survives in sessionStorage.
const NEXT_KEY = 'telebeland.auth.next'

export function rememberNext(next: string) {
  try {
    sessionStorage.setItem(NEXT_KEY, safeNext(next))
  } catch {
    // Storage unavailable: the user simply lands on the dashboard.
  }
}

export function takeRememberedNext(): string {
  try {
    const value = sessionStorage.getItem(NEXT_KEY)
    sessionStorage.removeItem(NEXT_KEY)
    return safeNext(value)
  } catch {
    return APP_HOME
  }
}
