import { useEffect, useRef, useState } from 'react'
import { Clock3, Infinity as InfinityIcon } from 'lucide-react'
import { useI18n } from '@/i18n'
import { cn } from '@/lib/utils'
import { formatClock } from './useCountdown'

const WARN_MS = 5 * 60_000

/**
 * Remaining time from the server deadline. The visible clock is not a live
 * region (it would announce every second); thresholds are announced once.
 */
export function TimerPill({ remaining, compact = false }: { remaining: number | null; compact?: boolean }) {
  const { t, plural } = useI18n()
  const [announcement, setAnnouncement] = useState('')
  const announced = useRef(new Set<number>())

  useEffect(() => {
    if (remaining === null) return
    for (const threshold of [5, 1]) {
      if (remaining <= threshold * 60_000 && remaining > 0 && !announced.current.has(threshold)) {
        announced.current.add(threshold)
        setAnnouncement(`${t.attempt.timer.label}: ${plural(threshold, t.quiz.minutes)}`)
      }
    }
  }, [remaining, plural, t])

  if (remaining === null) {
    return (
      <span
        className="inline-flex h-9 items-center gap-1.5 rounded-full border border-line bg-surface px-3 text-sm text-ink-muted"
        title={t.attempt.timer.unlimitedLong}
      >
        <InfinityIcon className="size-4" aria-hidden />
        <span className={cn(compact && 'sr-only')}>{t.attempt.timer.unlimited}</span>
        {compact && <span aria-hidden>—</span>}
      </span>
    )
  }

  const warn = remaining <= WARN_MS
  return (
    <span
      role="timer"
      aria-label={`${t.attempt.timer.label}: ${formatClock(remaining)}`}
      className={cn(
        'inline-flex h-9 items-center gap-1.5 rounded-full border px-3 text-sm font-semibold',
        warn ? 'border-terracotta/45 bg-terracotta-soft text-terracotta-ink' : 'border-line bg-surface text-ink',
      )}
    >
      <Clock3 className={cn('size-4', warn ? 'text-terracotta-ink' : 'text-ink-muted')} aria-hidden />
      <span className="tabular min-w-[3.25rem] text-center" aria-hidden>
        {formatClock(remaining)}
      </span>
      {warn && !compact && <span className="font-medium">· {t.attempt.timer.endsSoon}</span>}
      <span className="sr-only" aria-live="assertive">
        {announcement}
      </span>
    </span>
  )
}
