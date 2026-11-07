import type { ComponentProps, ReactNode } from 'react'
import { AlertCircle, RotateCw, WifiOff } from 'lucide-react'
import { isApiError } from '@/api/errors'
import { useT, type Messages } from '@/i18n'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { EmptySheetIllustration } from '@/components/brand/Illustrations'
import { Skeleton } from '@/components/ui/skeleton'

export function errorMessage(t: Messages, error: unknown): string {
  if (!isApiError(error)) return t.errors.unknown
  switch (error.kind) {
    case 'network':
      return t.errors.network
    case 'unauthorized':
      return t.errors.unauthorized
    case 'forbidden':
    case 'csrf':
      return t.errors.forbidden
    case 'not_found':
      return t.errors.notFound
    case 'conflict':
    case 'answer_locked':
      return t.errors.conflict
    case 'validation':
      return t.errors.validation
    case 'too_large':
      return t.errors.tooLarge
    case 'unsupported':
      return t.errors.unsupported
    case 'unprocessable':
      return t.errors.unprocessable
    case 'rate_limited':
      return t.errors.rateLimited
    case 'unavailable':
      return t.errors.unavailable
    default:
      return t.errors.server
  }
}

export function EmptyState({
  title,
  text,
  action,
  illustration = true,
  className,
}: {
  title: string
  text?: string
  action?: ReactNode
  illustration?: boolean
  className?: string
}) {
  return (
    <div className={cn('flex flex-col items-center px-6 py-10 text-center', className)}>
      {illustration && <EmptySheetIllustration className="mb-4" />}
      <h3 className="font-display text-lg font-semibold text-ink">{title}</h3>
      {text && <p className="mt-1.5 max-w-sm text-[0.9375rem] leading-relaxed text-ink-muted">{text}</p>}
      {action && <div className="mt-5 flex flex-wrap justify-center gap-2">{action}</div>}
    </div>
  )
}

export function ErrorState({
  title,
  error,
  onRetry,
  retrying,
  className,
}: {
  title: string
  error: unknown
  onRetry?: () => void
  retrying?: boolean
  className?: string
}) {
  const t = useT()
  const offline = isApiError(error, 'network')
  const Icon = offline ? WifiOff : AlertCircle
  return (
    <div role="alert" className={cn('flex flex-col items-center px-6 py-10 text-center', className)}>
      <span className="mb-3 grid size-11 place-items-center rounded-full bg-danger-soft text-danger">
        <Icon className="size-5" aria-hidden />
      </span>
      <h3 className="font-display text-lg font-semibold text-ink">{title}</h3>
      <p className="mt-1.5 max-w-sm text-[0.9375rem] text-ink-muted">{errorMessage(t, error)}</p>
      {onRetry && (
        <Button variant="secondary" className="mt-5" onClick={onRetry} loading={retrying}>
          {!retrying && <RotateCw aria-hidden />}
          {t.common.retry}
        </Button>
      )}
    </div>
  )
}

export function SectionHeader({
  title,
  hint,
  action,
  id,
}: {
  title: string
  hint?: string
  action?: ReactNode
  id?: string
}) {
  return (
    <div className="mb-3 flex items-end justify-between gap-4">
      <div className="min-w-0">
        <h2 id={id} className="font-display text-xl font-semibold text-ink">
          {title}
        </h2>
        {hint && <p className="mt-0.5 text-sm text-ink-muted">{hint}</p>}
      </div>
      {action}
    </div>
  )
}

export function Panel({ className, children, ...props }: ComponentProps<'div'>) {
  return (
    <div className={cn('rounded-lg border border-line bg-surface', className)} {...props}>
      {children}
    </div>
  )
}

/** Top of a regular page: serif title, one supporting line, actions on the right. */
export function PageHeader({
  title,
  subtitle,
  actions,
  eyebrow,
}: {
  title: ReactNode
  subtitle?: ReactNode
  actions?: ReactNode
  eyebrow?: ReactNode
}) {
  return (
    <header className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between xl:gap-8">
      <div className="min-w-0">
        {eyebrow && <div className="mb-2 text-sm text-ink-subtle">{eyebrow}</div>}
        <h1 className="font-display text-[1.75rem] leading-tight font-semibold text-pretty text-ink [overflow-wrap:anywhere] md:text-[2rem]">
          {title}
        </h1>
        {subtitle && <div className="mt-2 max-w-2xl text-reading text-ink-muted">{subtitle}</div>}
      </div>
      {actions && <div className="flex shrink-0 flex-wrap gap-2.5">{actions}</div>}
    </header>
  )
}

/** Standard page width and gutters inside the app shell. */
export function PageContainer({ className, children }: { className?: string; children: ReactNode }) {
  return <div className={cn('mx-auto max-w-content px-4 pt-6 md:px-8 md:pt-10 lg:px-10', className)}>{children}</div>
}

export function ListSkeleton({ rows, compact = false }: { rows: number; compact?: boolean }) {
  return (
    <ul className="divide-y divide-line-soft" aria-hidden>
      {Array.from({ length: rows }, (_, i) => (
        <li key={i} className="flex items-center gap-4 px-5 py-4">
          <Skeleton className="size-10 rounded-md" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-4 w-3/5" />
            <Skeleton className="h-3 w-2/5" />
          </div>
          {!compact && <Skeleton className="h-9 w-24" />}
        </li>
      ))}
    </ul>
  )
}
