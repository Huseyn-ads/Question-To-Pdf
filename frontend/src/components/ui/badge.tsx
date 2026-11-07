import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

// Badges always carry text (and usually an icon), so state never relies on
// colour alone.
const badgeVariants = cva(
  "inline-flex w-fit shrink-0 items-center gap-1 rounded-sm px-2 py-0.5 text-xs font-medium whitespace-nowrap [&>svg]:size-3.5 [&>svg]:shrink-0",
  {
    variants: {
      tone: {
        neutral: 'bg-sunken text-ink-muted',
        forest: 'bg-forest-soft text-forest',
        sage: 'bg-sage-soft text-sage-ink',
        terracotta: 'bg-terracotta-soft text-terracotta-ink',
        danger: 'bg-danger-soft text-danger',
        warning: 'bg-warning-soft text-warning-ink',
        outline: 'border border-line text-ink-muted',
      },
    },
    defaultVariants: { tone: 'neutral' },
  },
)

function Badge({
  className,
  tone,
  ...props
}: React.ComponentProps<'span'> & VariantProps<typeof badgeVariants>) {
  return <span data-slot="badge" className={cn(badgeVariants({ tone }), className)} {...props} />
}

export { Badge, badgeVariants }
