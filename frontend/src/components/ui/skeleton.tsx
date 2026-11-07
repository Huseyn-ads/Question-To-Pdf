import type { ComponentProps } from 'react'
import { cn } from '@/lib/utils'

/**
 * Loading placeholder. A block-level <span>, so it is valid inside headings
 * and paragraphs that are still waiting for their text.
 */
function Skeleton({ className, ...props }: ComponentProps<'span'>) {
  return (
    <span
      data-slot="skeleton"
      aria-hidden
      className={cn('block animate-pulse rounded-md bg-sunken', className)}
      {...props}
    />
  )
}

export { Skeleton }
