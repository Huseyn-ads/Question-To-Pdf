import { useState } from 'react'
import { Link, useLocation } from 'react-router'
import { MailOpen, TriangleAlert } from 'lucide-react'
import { useT } from '@/i18n'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Field } from '@/components/common/forms'
import { AuthLayout } from './AuthLayout'
import { ResendVerification } from './parts'
import type { CheckEmailState } from './RegisterPage'

/** "Check your inbox" after registration. Verification itself happens on the backend page from the email. */
export function CheckEmailPage() {
  const t = useT()
  const location = useLocation()
  const state = location.state as CheckEmailState | null
  const [email, setEmail] = useState(state?.email ?? '')

  return (
    <AuthLayout title={t.auth.checkTitle}>
      <div className="flex flex-col gap-6">
        {state ? (
          state.sent ? (
            <div className="flex gap-4">
              <span className="grid size-11 shrink-0 place-items-center rounded-full bg-forest-soft text-forest">
                <MailOpen className="size-5" aria-hidden />
              </span>
              <p className="text-reading text-ink [overflow-wrap:anywhere]">{t.auth.checkSent(state.email)}</p>
            </div>
          ) : (
            <Alert tone="warning">
              <TriangleAlert aria-hidden />
              <AlertDescription className="[overflow-wrap:anywhere]">{t.auth.checkNotSent(state.email)}</AlertDescription>
            </Alert>
          )
        ) : (
          <Field label={t.auth.email}>
            {(control) => (
              <Input
                {...control}
                type="email"
                autoComplete="email"
                value={email}
                onChange={(event) => setEmail(event.target.value.trim())}
              />
            )}
          </Field>
        )}
        <p className="text-[0.9375rem] text-ink-muted">
          {t.auth.checkAfter} {t.auth.checkSpam}
        </p>
        {/* A successful registration has just issued a token, so the 60 s cooldown already runs. */}
        <ResendVerification email={email} startCooling={state?.sent ?? false} />
        <Button asChild size="lg" className="w-full">
          <Link to="/login" state={{ email }}>
            {t.auth.toLogin}
          </Link>
        </Button>
      </div>
    </AuthLayout>
  )
}
