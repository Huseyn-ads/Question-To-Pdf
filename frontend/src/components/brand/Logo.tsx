import { cn } from '@/lib/utils'

/** Mark: an open notebook with a bookmark ribbon; wordmark set in the serif. */
export function LogoMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 32 32" aria-hidden className={cn('size-8 shrink-0', className)}>
      <rect width="32" height="32" rx="8" fill="#2F5141" />
      <path
        d="M8 10.5c2.6-.9 5.3-.6 8 1 2.7-1.6 5.4-1.9 8-1v12c-2.6-.9-5.3-.6-8 1-2.7-1.6-5.4-1.9-8-1z"
        fill="none"
        stroke="#FFFCF6"
        strokeWidth="1.6"
        strokeLinejoin="round"
      />
      <path d="M16 11.5v12" stroke="#FFFCF6" strokeWidth="1.6" strokeLinecap="round" />
      <path d="M19.5 9v6l1.5-1.2 1.5 1.2V9" fill="#D9895F" />
    </svg>
  )
}

export function Logo({ className, compact = false }: { className?: string; compact?: boolean }) {
  return (
    <span className={cn('inline-flex items-center gap-2.5', className)}>
      <LogoMark />
      {!compact && (
        <span className="font-display text-[1.375rem] leading-none font-semibold tracking-[-0.02em] text-ink">
          telebeland
        </span>
      )}
    </span>
  )
}
