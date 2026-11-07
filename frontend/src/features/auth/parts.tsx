import { useEffect, useState, type ComponentProps } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Eye, EyeOff, MailCheck } from 'lucide-react'
import { useApi } from '@/api/context'
import { isApiError } from '@/api/errors'
import { useT } from '@/i18n'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { errorMessage } from '@/components/common/states'

export function PasswordInput({ className, ...props }: Omit<ComponentProps<'input'>, 'type'>) {
  const t = useT()
  const [visible, setVisible] = useState(false)
  return (
    <div className="relative">
      <Input type={visible ? 'text' : 'password'} className={cn('pr-12', className)} {...props} />
      <button
        type="button"
        onClick={() => setVisible((v) => !v)}
        aria-pressed={visible}
        className="absolute top-1/2 right-1 grid size-9 -translate-y-1/2 place-items-center rounded-sm text-ink-muted hover:bg-sunken hover:text-ink"
      >
        {visible ? <EyeOff className="size-4.5" aria-hidden /> : <Eye className="size-4.5" aria-hidden />}
        <span className="sr-only">{visible ? t.auth.hidePassword : t.auth.showPassword}</span>
      </button>
    </div>
  )
}

const COOLDOWN_SECONDS = 60

/**
 * Resend button with the backend's 60 s cooldown mirrored on the client.
 * The 202 answer is deliberately vague (same text for every account state),
 * so the confirmation stays neutral too.
 */
export function ResendVerification({ email, startCooling = false }: { email: string; startCooling?: boolean }) {
  const t = useT()
  const api = useApi()
  const [cooldownUntil, setCooldownUntil] = useState(() => (startCooling ? Date.now() + COOLDOWN_SECONDS * 1000 : 0))
  const [now, setNow] = useState(() => Date.now())
  const resend = useMutation({
    mutationFn: () => api.auth.resendVerification(email),
    onSuccess: () => setCooldownUntil(Date.now() + COOLDOWN_SECONDS * 1000),
    onError: (error) => {
      if (isApiError(error, 'rate_limited')) {
        setCooldownUntil(Date.now() + (error.retryAfter ?? COOLDOWN_SECONDS) * 1000)
      }
    },
  })

  const secondsLeft = Math.max(0, Math.ceil((cooldownUntil - now) / 1000))
  useEffect(() => {
    if (secondsLeft <= 0) return
    const timer = window.setTimeout(() => setNow(Date.now()), 1000)
    return () => window.clearTimeout(timer)
  }, [secondsLeft, now])

  return (
    <div className="flex flex-col items-start gap-2">
      <Button
        type="button"
        variant="secondary"
        onClick={() => {
          setNow(Date.now())
          resend.mutate()
        }}
        loading={resend.isPending}
        disabled={secondsLeft > 0 || !email}
      >
        <MailCheck aria-hidden />
        {secondsLeft > 0 ? t.auth.resendIn(secondsLeft) : t.auth.resend}
      </Button>
      <p aria-live="polite" className="text-sm text-ink-muted">
        {resend.isSuccess && t.auth.resendDone}
        {resend.isError && <span className="text-danger">{errorMessage(t, resend.error)}</span>}
      </p>
    </div>
  )
}
