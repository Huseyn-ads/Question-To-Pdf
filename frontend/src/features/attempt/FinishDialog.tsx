import { useState } from 'react'
import { AlertTriangle, CheckCircle2 } from 'lucide-react'
import { useI18n } from '@/i18n'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { errorMessage } from '@/components/common/states'
import type { FinishOutcome } from './useAttemptSession'

const MAX_CHIPS = 24

export function FinishDialog({
  open,
  onOpenChange,
  unansweredNumbers,
  total,
  numberOf,
  onJump,
  onFinish,
  onRetrySaves,
  onFinished,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  unansweredNumbers: number[]
  total: number
  numberOf: (questionId: string) => number
  onJump: (number: number) => void
  onFinish: (options?: { ignoreUnsaved?: boolean }) => Promise<FinishOutcome>
  onRetrySaves: () => void
  onFinished: () => void
}) {
  const { t, plural } = useI18n()
  const [busy, setBusy] = useState(false)
  const [outcome, setOutcome] = useState<Exclude<FinishOutcome, { ok: true }> | null>(null)

  const run = async (ignoreUnsaved = false) => {
    setBusy(true)
    setOutcome(null)
    const result = await onFinish({ ignoreUnsaved })
    setBusy(false)
    if (result.ok) onFinished()
    else setOutcome(result)
  }

  const handleOpenChange = (next: boolean) => {
    if (busy) return
    if (!next) setOutcome(null)
    onOpenChange(next)
  }

  const unsaved = outcome?.reason === 'unsaved' ? outcome.failed.map(numberOf).sort((a, b) => a - b) : []
  const allAnswered = unansweredNumbers.length === 0

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent showCloseButton={!busy} className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t.attempt.finishDialog.title}</DialogTitle>
          <DialogDescription>
            {allAnswered
              ? t.attempt.finishDialog.allAnswered
              : t.attempt.finishDialog.someUnanswered(plural(unansweredNumbers.length, t.quiz.questions))}
          </DialogDescription>
        </DialogHeader>

        {allAnswered ? (
          <p className="flex items-center gap-2 text-sm font-medium text-forest">
            <CheckCircle2 className="size-5" aria-hidden />
            {t.attempt.answeredOf(total, total)}
          </p>
        ) : (
          <div>
            <p className="mb-2 text-sm font-medium text-ink">{t.attempt.finishDialog.jumpTo}</p>
            <ul className="flex flex-wrap gap-1.5">
              {unansweredNumbers.slice(0, MAX_CHIPS).map((n) => (
                <li key={n}>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => onJump(n)}
                    className="tabular h-8 min-w-9 rounded-sm border border-line bg-surface px-2 text-sm text-ink hover:border-control hover:bg-forest-tint"
                  >
                    {n}
                  </button>
                </li>
              ))}
              {unansweredNumbers.length > MAX_CHIPS && (
                <li className="self-center text-sm text-ink-muted">+{unansweredNumbers.length - MAX_CHIPS}</li>
              )}
            </ul>
          </div>
        )}

        {busy && (
          <p className="text-sm text-ink-muted" aria-live="polite">
            {t.attempt.finishDialog.waitingSaves}
          </p>
        )}

        {outcome?.reason === 'unsaved' && (
          <Alert tone="danger">
            <AlertTriangle aria-hidden />
            <AlertTitle>{t.attempt.finishDialog.saveFailed(unsaved.join(', '))}</AlertTitle>
            <AlertDescription>
              <p>{t.attempt.finishDialog.saveFailedHint}</p>
              <div className="flex flex-wrap gap-2">
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => {
                    setOutcome(null)
                    onRetrySaves()
                  }}
                >
                  {t.common.retry}
                </Button>
                <Button size="sm" variant="danger-ghost" onClick={() => void run(true)}>
                  {t.attempt.finishDialog.finishAnyway}
                </Button>
              </div>
            </AlertDescription>
          </Alert>
        )}
        {outcome?.reason === 'error' && (
          <Alert tone="danger">
            <AlertTriangle aria-hidden />
            <AlertTitle>{t.attempt.finishDialog.finishFailed}</AlertTitle>
            <AlertDescription>
              <p>{errorMessage(t, outcome.error)}</p>
            </AlertDescription>
          </Alert>
        )}

        <DialogFooter>
          <Button variant="secondary" disabled={busy} onClick={() => handleOpenChange(false)}>
            {t.attempt.finishDialog.keepGoing}
          </Button>
          <Button loading={busy} onClick={() => void run(false)}>
            {outcome?.reason === 'error' ? t.common.retry : t.attempt.finishDialog.confirm}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
