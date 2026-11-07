import { Navigate, Outlet, useLocation } from 'react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useT } from '@/i18n'
import { ErrorState } from '@/components/common/states'
import type { LoginLocationState } from './LoginPage'
import { getSessionEndReason, loginPath, useMe } from './session'

export function FullPageLoader() {
  const t = useT()
  return (
    <div role="status" className="grid min-h-dvh place-items-center text-ink-muted">
      <span className="flex items-center gap-3">
        <Loader2 className="size-5 animate-spin text-forest" aria-hidden />
        {t.common.loading}
      </span>
    </div>
  )
}

/** Route guard: profile → render, no session → login with a safe return URL. */
export function RequireAuth() {
  const t = useT()
  const me = useMe()
  const location = useLocation()
  const queryClient = useQueryClient()

  if (me.data) return <Outlet />
  if (me.data === null) {
    const reason = getSessionEndReason(queryClient)
    const state: LoginLocationState = { expired: reason === 'expired', loggedOut: reason === 'logout' }
    const target = reason === 'logout' ? '/login' : loginPath(`${location.pathname}${location.search}`)
    return <Navigate to={target} replace state={state} />
  }
  if (me.isError) {
    return (
      <div className="grid min-h-dvh place-items-center">
        <ErrorState
          title={t.errors.unknown}
          error={me.error}
          onRetry={() => void me.refetch()}
          retrying={me.isFetching}
        />
      </div>
    )
  }
  return <FullPageLoader />
}
