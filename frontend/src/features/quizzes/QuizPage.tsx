import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft,
  ArrowRight,
  CheckCircle2,
  ChevronDown,
  Clock3,
  ListOrdered,
  MoreHorizontal,
  Play,
  RotateCcw,
  Shuffle,
  TimerOff,
  Trash2,
} from 'lucide-react'
import { useApi } from '@/api/context'
import { isApiError } from '@/api/errors'
import { queryKeys } from '@/api/queryKeys'
import type { AttemptSummary } from '@/api/types'
import { usePaths } from '@/app/paths'
import { useI18n } from '@/i18n'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { ConfirmDialog } from '@/components/common/ConfirmDialog'
import { QuestionText } from '@/components/common/QuestionText'
import { ErrorState, ListSkeleton, PageContainer, PageHeader, Panel, SectionHeader, errorMessage } from '@/components/common/states'
import { useToast } from '@/components/common/toast'
import { optionLetter } from '@/features/attempt/optionLetter'
import { useBank } from '@/features/banks/hooks'
import { useAttemptHistory, useQuiz, useStartAttempt } from './hooks'

export function AttemptStatusBadge({ status }: { status: AttemptSummary['status'] }) {
  const { t } = useI18n()
  if (status === 'in_progress')
    return (
      <Badge tone="warning">
        <Clock3 aria-hidden />
        {t.quizzes.status.in_progress}
      </Badge>
    )
  if (status === 'expired')
    return (
      <Badge tone="terracotta">
        <TimerOff aria-hidden />
        {t.quizzes.status.expired}
      </Badge>
    )
  return (
    <Badge tone="forest">
      <CheckCircle2 aria-hidden />
      {t.quizzes.status.completed}
    </Badge>
  )
}

export function QuizPage() {
  const { quizId = '' } = useParams()
  const { t, plural } = useI18n()
  const api = useApi()
  const paths = usePaths()
  const toast = useToast()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const quiz = useQuiz(quizId)
  const bank = useBank(quiz.data?.question_bank_id)
  const history = useAttemptHistory(quizId)
  const start = useStartAttempt(quizId)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [showContents, setShowContents] = useState(false)

  const remove = useMutation({
    mutationFn: () => api.quizzes.remove(quizId),
    onSuccess: () => {
      toast.success(t.quizzes.deleted)
      queryClient.removeQueries({ queryKey: queryKeys.quiz(quizId) })
      queryClient.removeQueries({ queryKey: ['quizzes', 'history', quizId] })
      void queryClient.invalidateQueries({ queryKey: ['quizzes', 'list'] })
      void navigate(paths.quizzes, { replace: true })
    },
  })

  if (quiz.isError) {
    return (
      <PageContainer>
        <BackLink />
        <Panel className="mt-6">
          <ErrorState
            title={t.quizzes.quizLoadError}
            error={quiz.error}
            onRetry={isApiError(quiz.error, 'not_found') ? undefined : () => void quiz.refetch()}
            retrying={quiz.isFetching}
          />
        </Panel>
      </PageContainer>
    )
  }

  const attempts = history.data?.pages.flat() ?? []
  const latest = attempts[0]
  const active = latest?.status === 'in_progress' ? latest : undefined
  const title = bank.data?.title ?? (bank.isError ? t.quizzes.bankMissing : undefined)

  return (
    <PageContainer>
      <BackLink />
      <div className="mt-4">
        <PageHeader
          eyebrow={
            quiz.data && bank.data ? (
              <Link to={paths.bank(bank.data.id)} className="hover:text-forest hover:underline hover:underline-offset-4">
                {t.quizzes.fromBank}
              </Link>
            ) : undefined
          }
          title={title ?? <Skeleton className="h-9 w-72" />}
          subtitle={
            quiz.data ? (
              <ul className="flex flex-wrap gap-x-5 gap-y-1.5 text-[0.9375rem]">
                <li className="flex items-center gap-1.5">
                  <ListOrdered className="size-4 text-ink-subtle" aria-hidden />
                  {plural(quiz.data.question_count, t.quiz.questions)}
                </li>
                <li className="flex items-center gap-1.5">
                  {quiz.data.time_limit_minutes ? (
                    <Clock3 className="size-4 text-ink-subtle" aria-hidden />
                  ) : (
                    <TimerOff className="size-4 text-ink-subtle" aria-hidden />
                  )}
                  {quiz.data.time_limit_minutes
                    ? plural(quiz.data.time_limit_minutes, t.quiz.minutes)
                    : t.quiz.noTimeLimit}
                </li>
                {quiz.data.shuffle_options && (
                  <li className="flex items-center gap-1.5">
                    <Shuffle className="size-4 text-ink-subtle" aria-hidden />
                    {t.quiz.shuffled}
                  </li>
                )}
              </ul>
            ) : (
              <Skeleton className="h-5 w-64" />
            )
          }
          actions={
            <>
              {active ? (
                <Button asChild size="lg">
                  <Link to={paths.attempt(active.attempt_id)}>
                    {t.quizzes.continue}
                    <ArrowRight aria-hidden />
                  </Link>
                </Button>
              ) : (
                <Button
                  size="lg"
                  onClick={() => start.mutate()}
                  loading={start.isPending}
                  disabled={!quiz.data || history.isPending}
                >
                  {latest ? <RotateCcw aria-hidden /> : <Play aria-hidden />}
                  {latest ? t.quizzes.retake : t.quizzes.start}
                </Button>
              )}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon" className="size-12" aria-label={t.common.actions}>
                    <MoreHorizontal aria-hidden />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem variant="destructive" onSelect={() => setDeleteOpen(true)}>
                    <Trash2 aria-hidden />
                    {t.quizzes.delete}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          }
        />
        {start.isError && (
          <p role="alert" className="mt-3 text-sm text-danger">
            {errorMessage(t, start.error)}
          </p>
        )}
      </div>

      <div className="mt-10 grid gap-10 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)] lg:gap-8">
        <section aria-labelledby="history-title">
          <SectionHeader id="history-title" title={t.quizzes.history} />
          <Panel>
            {history.isPending ? (
              <ListSkeleton rows={3} compact />
            ) : history.isError ? (
              <ErrorState
                title={t.quizzes.historyLoadError}
                error={history.error}
                onRetry={() => void history.refetch()}
                retrying={history.isFetching}
              />
            ) : attempts.length === 0 ? (
              <p className="px-5 py-8 text-center text-ink-muted">{t.quizzes.historyEmpty}</p>
            ) : (
              <ul className="divide-y divide-line-soft">
                {attempts.map((attempt) => (
                  <HistoryRow key={attempt.attempt_id} attempt={attempt} />
                ))}
              </ul>
            )}
          </Panel>
          {history.hasNextPage && (
            <div className="mt-3 flex justify-center">
              <Button
                variant="secondary"
                onClick={() => void history.fetchNextPage()}
                loading={history.isFetchingNextPage}
              >
                {t.common.showMore}
              </Button>
            </div>
          )}
        </section>

        <section aria-labelledby="contents-title">
          <SectionHeader id="contents-title" title={t.quizzes.contents} hint={t.quizzes.contentsHint} />
          <Panel>
            <button
              type="button"
              onClick={() => setShowContents((v) => !v)}
              aria-expanded={showContents}
              aria-controls="quiz-contents"
              className="flex w-full items-center justify-between gap-3 px-5 py-4 text-left font-medium text-ink hover:bg-forest-tint/60"
            >
              {showContents ? t.quizzes.hideContents : t.quizzes.showContents}
              <ChevronDown
                className={cn('size-5 text-ink-subtle transition-transform', showContents && 'rotate-180')}
                aria-hidden
              />
            </button>
            {showContents && (
              <ol id="quiz-contents" className="divide-y divide-line-soft border-t border-line-soft">
                {(quiz.data?.questions ?? []).map((question, index) => (
                  <li key={question.id} className="flex gap-3 px-5 py-4">
                    <span className="tabular w-6 shrink-0 text-sm font-semibold text-ink-subtle">{index + 1}.</span>
                    <div className="min-w-0 flex-1">
                      <QuestionText text={question.text} />
                      <ul className="mt-2 grid gap-1 text-[0.9375rem] text-ink-muted">
                        {question.options.map((option, optionIndex) => (
                          <li key={option.id} className="flex gap-2 [overflow-wrap:anywhere]">
                            <span className="w-4 shrink-0 font-semibold text-ink-subtle">{optionLetter(optionIndex)}</span>
                            {option.text}
                          </li>
                        ))}
                      </ul>
                    </div>
                  </li>
                ))}
              </ol>
            )}
          </Panel>
        </section>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={(open) => {
          setDeleteOpen(open)
          if (!open) remove.reset()
        }}
        title={t.quizzes.deleteTitle}
        text={t.quizzes.deleteText}
        confirmLabel={t.common.delete}
        onConfirm={() => remove.mutate()}
        pending={remove.isPending}
        error={remove.error}
      />
    </PageContainer>
  )
}

function BackLink() {
  const { t } = useI18n()
  const paths = usePaths()
  return (
    <Link
      to={paths.quizzes}
      className="inline-flex items-center gap-1.5 rounded-sm text-sm font-medium text-ink-muted hover:text-forest"
    >
      <ArrowLeft className="size-4" aria-hidden />
      {t.quizzes.title}
    </Link>
  )
}

function HistoryRow({ attempt }: { attempt: AttemptSummary }) {
  const { t, formatDate, formatNumber } = useI18n()
  const paths = usePaths()
  const started = formatDate(attempt.started_at, { day: 'numeric', month: 'long', hour: '2-digit', minute: '2-digit' })
  const inProgress = attempt.status === 'in_progress'
  return (
    <li className="flex flex-wrap items-center gap-x-4 gap-y-2 px-4 py-3.5 sm:px-5">
      <div className="min-w-0 flex-1">
        <p className="text-[0.9375rem] font-medium text-ink">{t.quizzes.attemptStarted(started)}</p>
        <div className="mt-1 flex flex-wrap items-center gap-2">
          <AttemptStatusBadge status={attempt.status} />
          {!inProgress && attempt.percentage !== null && (
            <span className="tabular text-sm text-ink-muted">
              {t.quizzes.score(attempt.correct_answers, attempt.gradable_count)} ·{' '}
              <b className="font-semibold text-ink">
                {formatNumber(attempt.percentage / 100, { style: 'percent', maximumFractionDigits: 1 })}
              </b>
            </span>
          )}
        </div>
      </div>
      <Button asChild variant={inProgress ? 'soft' : 'secondary'} size="sm">
        <Link to={inProgress ? paths.attempt(attempt.attempt_id) : paths.result(attempt.attempt_id)}>
          {inProgress ? t.dashboard.continueQuiz : t.quizzes.openResult}
        </Link>
      </Button>
    </li>
  )
}
