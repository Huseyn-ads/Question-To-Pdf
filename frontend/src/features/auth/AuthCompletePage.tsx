import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useApi } from '@/api/context'
import { isApiError } from '@/api/errors'
import { clearCsrfToken } from '@/api/http'
import { useT } from '@/i18n'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/components/common/states'
import { AuthLayout } from './AuthLayout'
import { beginSession, takeRememberedNext } from './session'

/**
 * Landing page after the Google callback (backend 303 → /auth/complete).
 * The new session cookie is already set; load the profile like after a
 * password login, forget any previous user's cache and return to the app.
 */
export function AuthCompletePage() {
  const t = useT()
  const api = useApi()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const started = useRef(false)
  const [failure, setFailure] = useState<unknown>(null)

  useEffect(() => {
    if (started.current) return
    started.current = true
    clearCsrfToken()
    api.auth
      .me()
      .then((user) => {
        beginSession(queryClient, user)
        void navigate(takeRememberedNext(), { replace: true })
      })
      .catch((error: unknown) => setFailure(error))
  }, [api, navigate, queryClient])

  if (failure == null) {
    return (
      <AuthLayout title={t.auth.completeTitle}>
        <p role="status" className="flex items-center gap-3 text-ink-muted">
          <Loader2 className="size-5 animate-spin text-forest" aria-hidden />
          {t.common.loading}
        </p>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout title={t.auth.completeFailedTitle}>
      <p role="alert" className="text-reading text-ink-muted">
        {isApiError(failure, 'unauthorized') ? t.auth.completeFailedText : errorMessage(t, failure)}
      </p>
      <Button asChild size="lg" className="mt-6 w-full">
        <Link to="/login">{t.auth.toLogin}</Link>
      </Button>
    </AuthLayout>
  )
}
