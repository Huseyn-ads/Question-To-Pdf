import { Link } from 'react-router'
import { useQuery } from '@tanstack/react-query'
import { ClipboardList, Clock3 } from 'lucide-react'
import { useApi } from '@/api/context'
import { queryKeys } from '@/api/queryKeys'
import type { AttemptSummary, QuestionBank, QuizSummary } from '@/api/types'
import { usePaths } from '@/app/paths'
import { useI18n } from '@/i18n'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { errorMessage } from '@/components/common/states'
import { useStartAttempt } from './hooks'

export interface QuizRowItem {
  quiz: QuizSummary
  bank: QuestionBank | undefined
  /** Latest attempt (history is sorted started_at DESC); null = never taken; undefined = loading/failed. */
  lastAttempt: AttemptSummary | null | undefined
}

/** Reads the latest attempt itself (history limit=1, shared cache with the dashboard). */
export function QuizRowWithHistory({ quiz, bank }: { quiz: QuizSummary; bank: QuestionBank | undefined }) {
  const api = useApi()
  const history = useQuery({
    queryKey: queryKeys.history(quiz.id, { limit: 1 }),
    queryFn: ({ signal }) => api.attempts.history(quiz.id, { limit: 1 }, signal),
  })
  return <QuizRow item={{ quiz, bank, lastAttempt: history.data ? (history.data[0] ?? null) : undefined }} />
}

export function ResultCell({ attempt }: { attempt: AttemptSummary | null | undefined }) {
  const { t, formatNumber } = useI18n()
  if (attempt === undefined) return <Skeleton className="h-8 w-20" />
  if (attempt === null) return <span className="text-sm text-ink-subtle">{t.dashboard.notTaken}</span>
  if (attempt.status === 'in_progress' || attempt.percentage === null) {
    return (
      <Badge tone="warning">
        <Clock3 aria-hidden />
        {t.dashboard.inProgress}
      </Badge>
    )
  }
  const pct = attempt.percentage
  return (
    <span className="flex flex-col items-start gap-1 @lg:items-end">
      <span className="text-xs whitespace-nowrap text-ink-subtle">{t.dashboard.lastResult}</span>
      <span className="flex items-center gap-2">
        <span className="h-1.5 w-14 overflow-hidden rounded-full bg-sunken" aria-hidden>
          <span className="block h-full rounded-full bg-sage" style={{ width: `${Math.min(100, Math.max(0, pct))}%` }} />
        </span>
        <span className="tabular text-[0.9375rem] font-semibold text-ink">
          {formatNumber(pct / 100, { style: 'percent', maximumFractionDigits: 1 })}
        </span>
      </span>
    </span>
  )
}

export function QuizRow({ item }: { item: QuizRowItem }) {
  const { t, plural, formatRelativeDay } = useI18n()
  const paths = usePaths()
  const start = useStartAttempt(item.quiz.id)
  const title = item.bank?.title ?? t.quiz.untitled
  const last = item.lastAttempt
  const inProgress = last?.status === 'in_progress'
  const meta = [
    plural(item.quiz.question_count, t.quiz.questions),
    item.quiz.time_limit_minutes ? t.quiz.minutesShort(item.quiz.time_limit_minutes) : t.quiz.noTimeLimit,
    t.dashboard.created(formatRelativeDay(item.quiz.created_at)),
  ]

  return (
    <li className="@container">
      {/* Laid out by the width of the list, not the viewport: the row sits in columns of different widths. */}
      <div className="flex flex-col gap-3 px-4 py-4 @lg:flex-row @lg:flex-wrap @lg:items-center @lg:gap-5 @lg:px-5">
        <div className="flex min-w-0 flex-1 items-start gap-4 @lg:min-w-64">
          <span className="grid size-10 shrink-0 place-items-center rounded-md bg-forest-soft text-forest">
            <ClipboardList className="size-5" aria-hidden />
          </span>
          <div className="min-w-0">
            <Link
              to={paths.quiz(item.quiz.id)}
              className="line-clamp-2 font-medium text-ink [overflow-wrap:anywhere] hover:text-forest hover:underline hover:decoration-forest/40 hover:underline-offset-4"
            >
              {title}
            </Link>
            <p className="mt-0.5 text-sm text-ink-subtle">{meta.join(' · ')}</p>
          </div>
        </div>
        <div className="flex items-center justify-between gap-3 @lg:justify-end @lg:gap-4">
          <ResultCell attempt={last} />
          {inProgress && last ? (
            <Button asChild variant="soft" className="@lg:min-w-32">
              <Link to={paths.attempt(last.attempt_id)}>{t.dashboard.continueQuiz}</Link>
            </Button>
          ) : (
            <Button
              variant="secondary"
              className="@lg:min-w-32"
              loading={start.isPending}
              disabled={last === undefined}
              onClick={() => start.mutate()}
            >
              {last ? t.dashboard.retakeQuiz : t.dashboard.takeQuiz}
            </Button>
          )}
        </div>
        {start.isError && (
          <p role="alert" className="text-sm text-danger @lg:basis-full">
            {errorMessage(t, start.error)}
          </p>
        )}
      </div>
    </li>
  )
}

