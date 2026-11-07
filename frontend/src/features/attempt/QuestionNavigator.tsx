import { AlertTriangle } from 'lucide-react'
import type { SnapshotQuestion } from '@/api/types'
import { useT } from '@/i18n'
import { cn } from '@/lib/utils'
import type { SaverSnapshot } from './answerSaver'

export function QuestionNavigator({
  questions,
  currentIndex,
  selections,
  saves,
  onJump,
  columns = 5,
}: {
  questions: SnapshotQuestion[]
  currentIndex: number
  selections: Record<string, string>
  saves: SaverSnapshot
  onJump: (index: number) => void
  columns?: 5 | 6
}) {
  const t = useT()
  return (
    <div>
      <ol className={cn('grid gap-2', columns === 5 ? 'grid-cols-5' : 'grid-cols-6')}>
        {questions.map((question, index) => {
          const answered = Boolean(selections[question.id])
          const failed = saves.questions[question.id]?.status === 'error'
          const current = index === currentIndex
          const state = failed
            ? t.attempt.stateError
            : answered
              ? t.attempt.stateAnswered
              : t.attempt.stateUnanswered
          return (
            <li key={question.id}>
              <button
                type="button"
                onClick={() => onJump(index)}
                aria-current={current ? 'step' : undefined}
                aria-label={t.attempt.navigatorItem(index + 1, current ? `${state}, ${t.attempt.stateCurrent}` : state)}
                className={cn(
                  'relative grid h-10 w-full place-items-center rounded-sm border text-sm transition-colors tabular',
                  answered
                    ? 'border-forest/15 bg-forest-soft font-semibold text-forest hover:bg-[#d7e3d8]'
                    : 'border-line bg-surface text-ink-muted hover:border-control/70',
                  failed && 'border-danger bg-danger-soft text-danger',
                  current && 'ring-2 ring-ink ring-offset-2 ring-offset-surface',
                )}
              >
                {index + 1}
                {answered && !failed && (
                  <span aria-hidden className="absolute inset-x-2.5 bottom-1 h-0.5 rounded-full bg-forest/70" />
                )}
                {failed && <AlertTriangle aria-hidden className="absolute -top-1.5 -right-1.5 size-3.5 fill-surface" />}
              </button>
            </li>
          )
        })}
      </ol>
      <ul className="mt-4 flex flex-wrap gap-x-4 gap-y-1.5 text-xs text-ink-muted" aria-hidden>
        <li className="flex items-center gap-1.5">
          <span className="relative size-3.5 rounded-xs border border-forest/15 bg-forest-soft">
            <span className="absolute inset-x-0.5 bottom-0.5 h-px bg-forest" />
          </span>
          {t.attempt.legendAnswered}
        </li>
        <li className="flex items-center gap-1.5">
          <span className="size-3.5 rounded-xs border border-line bg-surface" />
          {t.attempt.legendUnanswered}
        </li>
        <li className="flex items-center gap-1.5">
          <span className="size-3.5 rounded-xs border border-line bg-surface ring-2 ring-ink ring-offset-1 ring-offset-surface" />
          {t.attempt.legendCurrent}
        </li>
      </ul>
    </div>
  )
}
