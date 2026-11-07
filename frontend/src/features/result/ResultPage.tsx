import { useMemo, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, Check, CheckCircle2, CircleSlash, RotateCcw, TimerOff, X, XCircle } from 'lucide-react'
import { useApi } from '@/api/context'
import { isApiError } from '@/api/errors'
import { queryKeys } from '@/api/queryKeys'
import type { AttemptResult } from '@/api/types'
import { usePaths } from '@/app/paths'
import { useI18n } from '@/i18n'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { QuestionText } from '@/components/common/QuestionText'
import { ErrorState, PageContainer, Panel, errorMessage } from '@/components/common/states'
import { optionLetter } from '@/features/attempt/optionLetter'
import { useBank } from '@/features/banks/hooks'
import { useQuiz, useStartAttempt } from '@/features/quizzes/hooks'
import { reviewItems, type ReviewItem, type Verdict } from './reviewItems'

type Filter = 'all' | Verdict

function formatPercent(formatNumber: ReturnType<typeof useI18n>['formatNumber'], value: number) {
  return formatNumber(value / 100, { style: 'percent', maximumFractionDigits: 1 })
}

export function ResultPage() {
  const { attemptId = '' } = useParams()
  const { t } = useI18n()
  const api = useApi()
  const paths = usePaths()
  const result = useQuery({
    queryKey: queryKeys.result(attemptId),
    queryFn: ({ signal }) => api.attempts.result(attemptId, signal),
    // A closed attempt's result never changes.
    staleTime: Infinity,
    retry: (count, error) => !isApiError(error, 'conflict') && !isApiError(error, 'not_found') && count < 2,
  })

  // 409: the attempt is still running — the result does not exist yet.
  if (isApiError(result.error, 'conflict')) return <Navigate to={paths.attempt(attemptId)} replace />

  if (result.isError) {
    return (
      <PageContainer>
        <Panel>
          <ErrorState
            title={t.result.loadError}
            error={result.error}
            onRetry={isApiError(result.error, 'not_found') ? undefined : () => void result.refetch()}
            retrying={result.isFetching}
          />
        </Panel>
      </PageContainer>
    )
  }

  if (!result.data) {
    return (
      <PageContainer>
        <Skeleton className="h-6 w-40" />
        <Skeleton className="mt-6 h-56 rounded-lg" />
        <Skeleton className="mt-8 h-40 rounded-lg" />
      </PageContainer>
    )
  }

  return <ResultView result={result.data} />
}

function ResultView({ result }: { result: AttemptResult }) {
  const { t, plural, formatDate, formatNumber } = useI18n()
  const paths = usePaths()
  const quiz = useQuiz(result.quiz_id)
  const bank = useBank(quiz.data?.question_bank_id)
  const retake = useStartAttempt(result.quiz_id)
  const [filter, setFilter] = useState<Filter>('all')
  const items = useMemo(() => reviewItems(result), [result])
  const shown = filter === 'all' ? items : items.filter((item) => item.verdict === filter)

  const minutes = Math.round((Date.parse(result.finished_at) - Date.parse(result.started_at)) / 60_000)
  const duration = minutes < 1 ? t.result.durationUnderMinute : plural(minutes, t.result.durationMinutes)
  const total = Math.max(result.total_questions, 1)
  const counts: Record<Verdict, number> = {
    correct: result.correct_answers,
    wrong: result.wrong_answers,
    skipped: result.unanswered_questions,
  }
  const filters: { value: Filter; label: string; count: number }[] = [
    { value: 'all', label: t.result.filterAll, count: items.length },
    { value: 'wrong', label: t.result.filterWrong, count: counts.wrong },
    { value: 'skipped', label: t.result.filterSkipped, count: counts.skipped },
    { value: 'correct', label: t.result.filterCorrect, count: counts.correct },
  ]

  return (
    <PageContainer className="pb-12">
      <Link
        to={paths.quiz(result.quiz_id)}
        className="inline-flex items-center gap-1.5 rounded-sm text-sm font-medium text-ink-muted hover:text-forest"
      >
        <ArrowLeft className="size-4" aria-hidden />
        {t.result.toQuiz}
      </Link>

      <Panel className="mt-4 overflow-hidden">
        <div className="grid gap-8 p-5 sm:p-8 md:grid-cols-[auto_minmax(0,1fr)] md:items-center md:gap-12">
          <div>
            <p className="text-sm font-medium text-ink-muted">{t.result.scoreLabel}</p>
            <p className="tabular mt-1 font-display text-[3.5rem] leading-none font-semibold text-forest sm:text-[4.5rem]">
              {formatPercent(formatNumber, result.percentage)}
            </p>
            <p className="mt-2 text-[0.9375rem] text-ink">
              {t.result.scoreOf(result.correct_answers, result.total_questions)}
            </p>
          </div>
          <div className="min-w-0">
            <h1 className="font-display text-2xl leading-tight font-semibold text-ink [overflow-wrap:anywhere]">
              {bank.data?.title ?? (quiz.isPending || bank.isPending ? <Skeleton className="h-8 w-64" /> : t.result.title)}
            </h1>
            <p className="mt-2 flex items-start gap-2 text-[0.9375rem] text-ink-muted">
              {result.status === 'expired' ? (
                <TimerOff className="mt-0.5 size-4 shrink-0 text-terracotta-ink" aria-hidden />
              ) : (
                <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-forest" aria-hidden />
              )}
              {result.status === 'expired' ? t.result.statusExpired : t.result.statusCompleted}
            </p>
            <p className="mt-1 text-sm text-ink-subtle">
              {t.result.finishedAt(
                formatDate(result.finished_at, { day: 'numeric', month: 'long', hour: '2-digit', minute: '2-digit' }),
              )}{' '}
              · {t.result.duration(duration)}
            </p>

            {/* Proportion bar: each segment also has a labelled count below. */}
            <div className="mt-5 flex h-2.5 overflow-hidden rounded-full bg-sunken" aria-hidden>
              <span className="bg-forest" style={{ width: `${(counts.correct / total) * 100}%` }} />
              <span className="bg-terracotta" style={{ width: `${(counts.wrong / total) * 100}%` }} />
            </div>
            <ul className="mt-3 flex flex-wrap gap-x-6 gap-y-2 text-[0.9375rem]">
              <li className="flex items-center gap-2">
                <CheckCircle2 className="size-4 text-forest" aria-hidden />
                {t.result.correct} <b className="tabular font-semibold text-ink">{counts.correct}</b>
              </li>
              <li className="flex items-center gap-2">
                <XCircle className="size-4 text-terracotta-ink" aria-hidden />
                {t.result.wrong} <b className="tabular font-semibold text-ink">{counts.wrong}</b>
              </li>
              <li className="flex items-center gap-2">
                <CircleSlash className="size-4 text-ink-subtle" aria-hidden />
                {t.result.skipped} <b className="tabular font-semibold text-ink">{counts.skipped}</b>
              </li>
            </ul>
          </div>
        </div>
        <div className="flex flex-col gap-2.5 border-t border-line-soft bg-paper/60 px-5 py-4 sm:flex-row sm:px-8">
          <Button onClick={() => retake.mutate()} loading={retake.isPending}>
            <RotateCcw aria-hidden />
            {t.result.retake}
          </Button>
          <Button asChild variant="secondary">
            <Link to={paths.quiz(result.quiz_id)}>{t.result.toQuiz}</Link>
          </Button>
          <Button asChild variant="ghost">
            <Link to={paths.dashboard}>{t.result.toDashboard}</Link>
          </Button>
          {retake.isError && (
            <p role="alert" className="self-center text-sm text-danger">
              {errorMessage(t, retake.error)}
            </p>
          )}
        </div>
      </Panel>

      <section aria-labelledby="review-title" className="mt-10">
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <h2 id="review-title" className="font-display text-xl font-semibold text-ink">
            {t.result.review}
          </h2>
          <div
            role="radiogroup"
            aria-label={t.result.filterLabel}
            className="flex max-w-full overflow-x-auto rounded-md border border-line bg-surface p-1"
          >
            {filters.map((item) => (
              <button
                key={item.value}
                type="button"
                role="radio"
                aria-checked={filter === item.value}
                onClick={() => setFilter(item.value)}
                className={cn(
                  'h-8 shrink-0 rounded-sm px-3 text-sm font-medium whitespace-nowrap',
                  filter === item.value ? 'bg-forest-soft text-forest' : 'text-ink-muted hover:text-ink',
                )}
              >
                {item.label}
                <span className="tabular ml-1.5 text-xs">{item.count}</span>
              </button>
            ))}
          </div>
        </div>
        {shown.length === 0 ? (
          <p className="rounded-lg border border-line bg-surface px-5 py-8 text-center text-ink-muted">
            {t.result.filterEmpty}
          </p>
        ) : (
          <ol className="flex flex-col gap-3">
            {shown.map((item) => (
              <ReviewCard key={item.question.id} item={item} />
            ))}
          </ol>
        )}
      </section>
    </PageContainer>
  )
}

function VerdictBadge({ verdict }: { verdict: Verdict }) {
  const { t } = useI18n()
  if (verdict === 'correct')
    return (
      <Badge tone="forest">
        <CheckCircle2 aria-hidden />
        {t.result.questionCorrect}
      </Badge>
    )
  if (verdict === 'wrong')
    return (
      <Badge tone="terracotta">
        <XCircle aria-hidden />
        {t.result.questionWrong}
      </Badge>
    )
  return (
    <Badge tone="neutral">
      <CircleSlash aria-hidden />
      {t.result.questionSkipped}
    </Badge>
  )
}

function ReviewCard({ item }: { item: ReviewItem }) {
  const { t } = useI18n()
  const { question, verdict } = item
  return (
    <li
      className={cn(
        'rounded-lg border bg-surface p-4 sm:p-5',
        verdict === 'wrong' ? 'border-terracotta/45' : 'border-line',
      )}
    >
      <div className="flex flex-wrap items-center gap-3">
        <span className="tabular text-sm font-semibold text-ink-subtle">{t.attempt.questionLabel(question.position)}</span>
        <VerdictBadge verdict={verdict} />
      </div>
      <QuestionText text={question.text} as="h3" className="mt-2.5" />
      {verdict === 'skipped' && <p className="mt-2 text-sm text-ink-muted">{t.result.noAnswer}</p>}
      <ul className="mt-3 grid gap-1.5">
        {question.options.map((option, index) => {
          const selected = option.is_selected
          const correct = option.is_correct
          return (
            <li
              key={option.id}
              className={cn(
                'flex items-start gap-2.5 rounded-md border px-2.5 py-2 text-[0.9375rem]',
                correct && 'border-forest/40 bg-forest-tint',
                selected && !correct && 'border-terracotta/50 bg-terracotta-soft',
                !correct && !selected && 'border-transparent text-ink-muted',
              )}
            >
              <span
                className={cn(
                  'grid size-6 shrink-0 place-items-center rounded-sm text-xs font-semibold',
                  correct ? 'bg-forest text-surface' : selected ? 'bg-terracotta-ink text-surface' : 'bg-sunken text-ink-muted',
                )}
                aria-hidden
              >
                {correct ? <Check className="size-3.5" /> : selected ? <X className="size-3.5" /> : optionLetter(index)}
              </span>
              <span className="min-w-0 flex-1 text-ink [overflow-wrap:anywhere]">{option.text}</span>
              {(selected || correct) && (
                <span
                  className={cn(
                    'shrink-0 pt-0.5 text-xs font-semibold',
                    correct ? 'text-forest' : 'text-terracotta-ink',
                  )}
                >
                  {selected && correct
                    ? `${t.result.yourAnswer} · ${t.result.questionCorrect.toLowerCase()}`
                    : selected
                      ? t.result.yourAnswer
                      : t.result.correctAnswer}
                </span>
              )}
            </li>
          )
        })}
      </ul>
    </li>
  )
}
