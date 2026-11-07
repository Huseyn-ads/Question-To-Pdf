import { useId, type ReactNode } from 'react'
import { AlertCircle } from 'lucide-react'
import { useT } from '@/i18n'
import { cn } from '@/lib/utils'

export interface FieldControlProps {
  id: string
  'aria-describedby': string | undefined
  'aria-invalid': true | undefined
}

/**
 * Label + control + hint + error. The control gets matching id,
 * aria-describedby and aria-invalid, and the error is text with an icon,
 * never just a red border.
 */
export function Field({
  label,
  hint,
  error,
  optional,
  className,
  children,
}: {
  label: ReactNode
  hint?: ReactNode
  error?: string
  optional?: boolean
  className?: string
  children: (control: FieldControlProps) => ReactNode
}) {
  const t = useT()
  const id = useId()
  const hintId = hint ? `${id}-hint` : undefined
  const errorId = error ? `${id}-error` : undefined
  const describedBy = [errorId, hintId].filter(Boolean).join(' ') || undefined
  return (
    <div className={cn('flex flex-col gap-1.5', className)}>
      <label htmlFor={id} className="text-sm font-medium text-ink">
        {label}
        {optional && <span className="ml-1.5 font-normal text-ink-subtle">({t.common.optional})</span>}
      </label>
      {children({ id, 'aria-describedby': describedBy, 'aria-invalid': error ? true : undefined })}
      {error && <FieldError id={errorId}>{error}</FieldError>}
      {hint && (
        <p id={hintId} className="text-sm text-ink-subtle">
          {hint}
        </p>
      )}
    </div>
  )
}

export function FieldError({ id, children, className }: { id?: string; children: ReactNode; className?: string }) {
  return (
    <p id={id} className={cn('flex items-start gap-1.5 text-sm text-danger', className)}>
      <AlertCircle className="mt-0.5 size-4 shrink-0" aria-hidden />
      <span>{children}</span>
    </p>
  )
}

/** Native checkbox with a large hit area and a description line. */
export function CheckboxRow({
  checked,
  onChange,
  label,
  hint,
  disabled,
  name,
}: {
  checked: boolean
  onChange: (checked: boolean) => void
  label: ReactNode
  hint?: ReactNode
  disabled?: boolean
  name?: string
}) {
  const id = useId()
  return (
    <div className="flex items-start gap-3">
      <input
        id={id}
        name={name}
        type="checkbox"
        className="mt-0.5 size-5 shrink-0 cursor-pointer rounded-xs accent-forest disabled:cursor-not-allowed"
        checked={checked}
        disabled={disabled}
        aria-describedby={hint ? `${id}-hint` : undefined}
        onChange={(event) => onChange(event.target.checked)}
      />
      <div className="min-w-0">
        <label htmlFor={id} className="cursor-pointer text-[0.9375rem] font-medium text-ink">
          {label}
        </label>
        {hint && (
          <p id={`${id}-hint`} className="mt-0.5 text-sm text-ink-subtle">
            {hint}
          </p>
        )}
      </div>
    </div>
  )
}
