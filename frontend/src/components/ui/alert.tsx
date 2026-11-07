import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

// Inline notices. Every tone is paired with an icon and a title in usage,
// so meaning is never carried by colour alone.
const alertVariants = cva(
  'relative grid w-full grid-cols-[0_1fr] items-start gap-y-1 rounded-md border px-4 py-3 text-sm has-[>svg]:grid-cols-[1.25rem_1fr] has-[>svg]:gap-x-3 [&>svg]:size-5 [&>svg]:translate-y-px',
  {
    variants: {
      tone: {
        info: 'border-line bg-surface text-ink [&>svg]:text-forest',
        success: 'border-forest/25 bg-forest-tint text-ink [&>svg]:text-forest',
        warning: 'border-warning-ink/25 bg-warning-soft text-ink [&>svg]:text-warning-ink',
        danger: 'border-danger/30 bg-danger-soft text-ink [&>svg]:text-danger',
      },
    },
    defaultVariants: { tone: 'info' },
  },
)

function Alert({
  className,
  tone,
  role,
  ...props
}: React.ComponentProps<'div'> & VariantProps<typeof alertVariants>) {
  return (
    <div
      data-slot="alert"
      role={role ?? (tone === 'danger' ? 'alert' : 'status')}
      className={cn(alertVariants({ tone }), className)}
      {...props}
    />
  )
}

function AlertTitle({ className, ...props }: React.ComponentProps<'div'>) {
  return <div className={cn('col-start-2 font-semibold text-ink', className)} {...props} />
}

function AlertDescription({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      className={cn('col-start-2 grid justify-items-start gap-2 text-ink-muted [&_p]:leading-relaxed', className)}
      {...props}
    />
  )
}

export { Alert, AlertTitle, AlertDescription }
