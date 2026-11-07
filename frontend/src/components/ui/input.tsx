import * as React from 'react'
import { cn } from '@/lib/utils'

export const fieldControlClass = [
  'w-full min-w-0 rounded-md border border-control/80 bg-surface px-3.5 text-base text-ink',
  'placeholder:text-ink-subtle transition-[border-color,box-shadow] duration-150',
  'hover:border-control focus-visible:border-forest focus-visible:outline-2 focus-visible:outline-offset-0 focus-visible:outline-forest/25',
  'disabled:cursor-not-allowed disabled:bg-sunken disabled:opacity-70',
  'aria-invalid:border-danger aria-invalid:focus-visible:outline-danger/25',
].join(' ')

function Input({ className, type, ...props }: React.ComponentProps<'input'>) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(fieldControlClass, 'h-11', className)}
      {...props}
    />
  )
}

export { Input }
