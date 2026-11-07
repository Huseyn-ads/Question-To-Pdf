import { useState } from 'react'
import { Link, Navigate, useLocation, useNavigate, useSearchParams } from 'react-router'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { AlertCircle, Info, MailWarning } from 'lucide-react'
import { useApi } from '@/api/context'
import { isApiError } from '@/api/errors'
import { useT } from '@/i18n'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Field } from '@/components/common/forms'
import { errorMessage } from '@/components/common/states'
import { AuthDivider, AuthLayout, GoogleButton } from './AuthLayout'
import { PasswordInput, ResendVerification } from './parts'
import { loginSchema, type LoginValues } from './schemas'
import { beginSession, rememberNext, safeNext, useMe } from './session'

export interface LoginLocationState {
  expired?: boolean
  loggedOut?: boolean
  email?: string
}

export function LoginPage() {
  const t = useT()
  const api = useApi()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const location = useLocation()
  const [searchParams] = useSearchParams()
  const next = safeNext(searchParams.get('next'))
  const state = (location.state ?? {}) as LoginLocationState
  const me = useMe()
  const [unverifiedEmail, setUnverifiedEmail] = useState<string | null>(null)

  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema(t)),
    defaultValues: { email: state.email ?? '', password: '' },
  })
  const login = useMutation({
    mutationFn: (values: LoginValues) => api.auth.login({ email: values.email, password: values.password }),
    onSuccess: (user) => {
      beginSession(queryClient, user)
      void navigate(next, { replace: true })
    },
    onError: (error, values) => {
      setUnverifiedEmail(isApiError(error, 'email_unverified') ? values.email : null)
      if (isApiError(error, 'invalid_credentials')) form.setFocus('password')
    },
  })

  // Already signed in (e.g. opened /login in a second tab).
  if (me.data && !login.isPending && !login.isSuccess) return <Navigate to={next} replace />

  const error = login.error
  const errors = form.formState.errors

  return (
    <AuthLayout title={t.auth.loginTitle} subtitle={t.auth.loginSubtitle}>
      {state.expired && !login.isError && (
        <Alert tone="warning" className="mb-6">
          <Info aria-hidden />
          <AlertDescription>{t.auth.sessionExpired}</AlertDescription>
        </Alert>
      )}
      {state.loggedOut && !login.isError && (
        <Alert tone="success" className="mb-6">
          <Info aria-hidden />
          <AlertDescription>{t.auth.loggedOut}</AlertDescription>
        </Alert>
      )}

      <GoogleButton href={api.auth.googleStartUrl} label={t.auth.google} onBeforeNavigate={() => rememberNext(next)} />
      <AuthDivider />

      <form
        noValidate
        className="flex flex-col gap-5"
        onSubmit={form.handleSubmit((values) => {
          if (!login.isPending) login.mutate(values)
        })}
      >
        {error && unverifiedEmail ? (
          <Alert tone="warning">
            <MailWarning aria-hidden />
            <AlertTitle>{t.auth.unverifiedTitle}</AlertTitle>
            <AlertDescription>
              <p>{t.auth.unverifiedText}</p>
              <ResendVerification email={unverifiedEmail} />
            </AlertDescription>
          </Alert>
        ) : error ? (
          <Alert tone="danger">
            <AlertCircle aria-hidden />
            <AlertDescription>
              {isApiError(error, 'invalid_credentials') ? t.auth.invalidCredentials : errorMessage(t, error)}
            </AlertDescription>
          </Alert>
        ) : null}

        <Field label={t.auth.email} error={errors.email?.message}>
          {(control) => (
            <Input
              {...control}
              type="email"
              autoComplete="email"
              inputMode="email"
              placeholder={t.auth.emailPlaceholder}
              {...form.register('email')}
            />
          )}
        </Field>
        <Field label={t.auth.password} error={errors.password?.message}>
          {(control) => <PasswordInput {...control} autoComplete="current-password" {...form.register('password')} />}
        </Field>
        <Button type="submit" size="lg" className="mt-1 w-full" loading={login.isPending}>
          {t.auth.submitLogin}
        </Button>
      </form>

      <p className="mt-8 text-center text-[0.9375rem] text-ink-muted">
        {t.auth.noAccount}{' '}
        <Link
          to={searchParams.get('next') ? `/register?next=${encodeURIComponent(next)}` : '/register'}
          className="font-medium text-forest underline decoration-forest/40 underline-offset-4 hover:decoration-forest"
        >
          {t.auth.toRegister}
        </Link>
      </p>
    </AuthLayout>
  )
}
