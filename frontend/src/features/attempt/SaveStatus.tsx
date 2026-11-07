import { AlertTriangle, Check, CloudUpload, Loader2 } from 'lucide-react'
import { useT } from '@/i18n'
import { cn } from '@/lib/utils'
import type { SaverSnapshot } from './answerSaver'

/** Save indicator: icon + text for each state, announced politely. */
export function SaveStatus({
  overall,
  onRetry,
  className,
}: {
  overall: SaverSnapshot['overall']
  onRetry: () => void
  className?: string
}) {
  const t = useT()
  return (
    <div className={cn('flex min-h-6 items-center gap-1.5 text-sm', className)} aria-live="polite">
      {overall === 'saving' && (
        <>
          <Loader2 className="size-4 animate-spin text-ink-muted" aria-hidden />
          <span className="text-ink-muted">{t.attempt.save.saving}</span>
        </>
      )}
      {overall === 'saved' && (
        <>
          <Check className="size-4 text-forest" aria-hidden />
          <span className="text-forest">{t.attempt.save.saved}</span>
        </>
      )}
      {overall === 'idle' && (
        <>
          <CloudUpload className="size-4 text-ink-subtle" aria-hidden />
          <span className="text-ink-subtle">{t.attempt.save.idle}</span>
        </>
      )}
      {overall === 'error' && (
        <>
          <AlertTriangle className="size-4 text-danger" aria-hidden />
          <span className="font-medium text-danger">{t.attempt.save.error}</span>
          <button
            type="button"
            onClick={onRetry}
            className="ml-1 rounded-sm font-medium text-forest underline decoration-forest/40 underline-offset-4 hover:decoration-forest"
          >
            {t.attempt.save.retry}
          </button>
        </>
      )}
    </div>
  )
}
