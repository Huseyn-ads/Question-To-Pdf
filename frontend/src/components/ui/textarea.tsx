import * as React from 'react'
import { cn } from '@/lib/utils'
import { fieldControlClass } from '@/components/ui/input'

function Textarea({ className, ...props }: React.ComponentProps<'textarea'>) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(fieldControlClass, 'min-h-24 py-2.5 leading-relaxed', className)}
      {...props}
    />
  )
}

export { Textarea }
