import { Link, Navigate, useNavigate, useSearchParams } from 'react-router'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation } from '@tanstack/react-query'
import { AlertCircle } from 'lucide-react'
import { useApi } from '@/api/context'
import { isApiError } from '@/api/errors'
import { useT } from '@/i18n'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Field } from '@/components/common/forms'
import { errorMessage } from '@/components/common/states'
import { AuthDivider, AuthLayout, GoogleButton } from './AuthLayout'
import { PasswordInput } from './parts'
import { registerSchema, type RegisterValues } from './schemas'
import { rememberNext, safeNext, useMe } from './session'

export interface CheckEmailState {
  email: string
  sent: boolean
}

export function RegisterPage() {
  const t = useT()
  const api = useApi()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const next = safeNext(searchParams.get('next'))
  const me = useMe()
  const form = useForm<RegisterValues>({
    resolver: zodResolver(registerSchema(t)),
    defaultValues: { name: '', email: '', password: '' },
  })
  // Registration has no idempotency key: never retried automatically.
  const register = useMutation({
    mutationFn: (values: RegisterValues) => api.auth.register(values),
    onSuccess: (response) => {
      const state: CheckEmailState = { email: response.email, sent: response.verification_email_sent }
      void navigate('/check-email', { state })
    },
    onError: (error) => {
      if (isApiError(error, 'conflict')) form.setFocus('email')
    },
  })

  if (me.data) return <Navigate to={next} replace />

  const errors = form.formState.errors
  const error = register.error

  return (
    <AuthLayout title={t.auth.registerTitle} subtitle={t.auth.registerSubtitle}>
      <GoogleButton
        href={api.auth.googleStartUrl}
        label={t.auth.googleRegister}
        onBeforeNavigate={() => rememberNext(next)}
      />
      <AuthDivider />
      <form
        noValidate
        className="flex flex-col gap-5"
        onSubmit={form.handleSubmit((values) => {
          if (!register.isPending) register.mutate(values)
        })}
      >
        {error && (
          <Alert tone="danger">
            <AlertCircle aria-hidden />
            <AlertDescription>
              {isApiError(error, 'conflict') ? (
                <p>
                  {t.auth.emailTaken}{' '}
                  <Link
                    to="/login"
                    state={{ email: form.getValues('email') }}
                    className="font-medium text-forest underline underline-offset-4"
                  >
                    {t.auth.toLogin}
                  </Link>
                </p>
              ) : (
                errorMessage(t, error)
              )}
            </AlertDescription>
          </Alert>
        )}
        <Field label={t.auth.name} error={errors.name?.message}>
          {(control) => (
            <Input {...control} autoComplete="name" placeholder={t.auth.namePlaceholder} {...form.register('name')} />
          )}
        </Field>
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
        <Field label={t.auth.password} hint={t.auth.passwordHint} error={errors.password?.message}>
          {(control) => <PasswordInput {...control} autoComplete="new-password" {...form.register('password')} />}
        </Field>
        <Button type="submit" size="lg" className="mt-1 w-full" loading={register.isPending}>
          {t.auth.submitRegister}
        </Button>
      </form>
      <p className="mt-8 text-center text-[0.9375rem] text-ink-muted">
        {t.auth.haveAccount}{' '}
        <Link
          to="/login"
          className="font-medium text-forest underline decoration-forest/40 underline-offset-4 hover:decoration-forest"
        >
          {t.auth.toLogin}
        </Link>
      </p>
    </AuthLayout>
  )
}
