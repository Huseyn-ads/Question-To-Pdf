import { Link } from 'react-router'
import { useQuery } from '@tanstack/react-query'
import {
  ArrowRight,
  BookOpenText,
  ChevronRight,
  FileUp,
  Lightbulb,
  PlayCircle,
  Plus,
} from 'lucide-react'
import { useApi } from '@/api/context'
import { queryKeys } from '@/api/queryKeys'
import { usePaths } from '@/app/paths'
import { useI18n } from '@/i18n'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { DeskIllustration } from '@/components/brand/Illustrations'
import { EmptyState, ErrorState, ListSkeleton, Panel, SectionHeader } from '@/components/common/states'
import { useCountdown } from '@/features/attempt/useCountdown'
import { QuizRow } from '@/features/quizzes/QuizRow'
import { useDashboardData, type DashboardQuiz } from './useDashboardData'

function greetingFor(hour: number) {
  if (hour < 5) return 'night' as const
  if (hour < 12) return 'morning' as const
  if (hour < 18) return 'day' as const
  return 'evening' as const
}

function firstName(name: string) {
  return name.trim().split(/\s+/)[0] ?? name
}

export function DashboardPage() {
  const { t, formatDate, formatRelativeDay } = useI18n()
  const paths = usePaths()
  const data = useDashboardData()
  const now = new Date()
  const name = data.me.data ? firstName(data.me.data.name) : ''
  const isEmpty = data.banks.data?.length === 0 && data.quizzes.data?.items.length === 0

  return (
    <div className="mx-auto max-w-content px-4 pt-6 md:px-8 md:pt-10 lg:px-10">
      {/* Greeting */}
      <header className="flex items-center gap-6">
        <div className="min-w-0 flex-1">
          <p className="text-sm text-ink-subtle first-letter:uppercase">
            {formatDate(now.toISOString(), { weekday: 'long', day: 'numeric', month: 'long' })}
          </p>
          <h1 className="mt-1 font-display text-[1.75rem] leading-tight font-semibold text-ink md:text-[2.125rem]">
            {name ? t.dashboard.greeting[greetingFor(now.getHours())](name) : <Skeleton className="h-9 w-64" />}
          </h1>
          <p className="mt-2 max-w-xl text-reading text-ink-muted">
            {isEmpty ? t.dashboard.subtitleEmpty : t.dashboard.subtitle}
          </p>
          <div className="mt-5 grid grid-cols-2 gap-2.5 sm:flex sm:flex-wrap">
            <Button asChild size="lg" className="px-3 text-[0.9375rem] sm:px-5 sm:text-base">
              <Link to={paths.importDocx}>
                <FileUp aria-hidden />
                {t.dashboard.actionImport}
              </Link>
            </Button>
            <Button asChild size="lg" variant="secondary" className="px-3 text-[0.9375rem] sm:px-5 sm:text-base">
              <Link to={paths.newQuiz}>
                <Plus aria-hidden />
                {t.dashboard.actionCreateQuiz}
              </Link>
            </Button>
          </div>
        </div>
        <DeskIllustration className="hidden w-60 shrink-0 md:block lg:w-72" />
      </header>

      {data.active && <ContinueBanner item={data.active} />}

      <div className="mt-10 grid gap-10 xl:grid-cols-[minmax(0,1fr)_20rem] xl:gap-10">
        <div className="flex min-w-0 flex-col gap-10">
          <section aria-labelledby="recent-quizzes">
            <SectionHeader
              id="recent-quizzes"
              title={t.dashboard.recentQuizzes}
              hint={t.dashboard.recentQuizzesHint}
              action={
                data.recent.length > 0 && (
                  <Button asChild variant="link" size="sm">
                    <Link to={paths.quizzes}>{t.common.allItems}</Link>
                  </Button>
                )
              }
            />
            <Panel>
              {data.quizzes.isPending ? (
                <ListSkeleton rows={3} />
              ) : data.quizzes.isError ? (
                <ErrorState
                  title={t.dashboard.loadError}
                  error={data.quizzes.error}
                  onRetry={() => void data.quizzes.refetch()}
                  retrying={data.quizzes.isFetching}
                />
              ) : data.recent.length === 0 ? (
                <EmptyState
                  title={t.dashboard.noQuizzesTitle}
                  text={t.dashboard.noQuizzesText}
                  action={
                    <Button asChild variant="secondary">
                      <Link to={paths.newQuiz}>
                        <Plus aria-hidden />
                        {t.dashboard.actionCreateQuiz}
                      </Link>
                    </Button>
                  }
                />
              ) : (
                <ul className="divide-y divide-line-soft">
                  {data.recent.map((item) => (
                    <QuizRow key={item.quiz.id} item={item} />
                  ))}
                </ul>
              )}
            </Panel>
          </section>

          <section aria-labelledby="banks">
            <SectionHeader
              id="banks"
              title={t.dashboard.banks}
              hint={t.dashboard.banksHint}
              action={
                (data.banks.data?.length ?? 0) > 0 && (
                  <Button asChild variant="link" size="sm">
                    <Link to={paths.banks}>{t.common.allItems}</Link>
                  </Button>
                )
              }
            />
            <Panel>
              {data.banks.isPending ? (
                <ListSkeleton rows={3} compact />
              ) : data.banks.isError ? (
                <ErrorState
                  title={t.dashboard.loadError}
                  error={data.banks.error}
                  onRetry={() => void data.banks.refetch()}
                  retrying={data.banks.isFetching}
                />
              ) : data.banks.data.length === 0 ? (
                <EmptyState
                  title={t.dashboard.noBanksTitle}
                  text={t.dashboard.noBanksText}
                  action={
                    <Button asChild>
                      <Link to={paths.importDocx}>
                        <FileUp aria-hidden />
                        {t.dashboard.actionImport}
                      </Link>
                    </Button>
                  }
                />
              ) : (
                <ul className="divide-y divide-line-soft">
                  {data.banks.data.slice(0, 5).map((bank) => (
                    <li key={bank.id}>
                      <Link
                        to={paths.bank(bank.id)}
                        className="group flex items-center gap-4 px-4 py-3.5 hover:bg-forest-tint/60 sm:px-5"
                      >
                        <span className="grid size-10 shrink-0 place-items-center rounded-md bg-sage-soft text-sage-ink">
                          <BookOpenText className="size-5" aria-hidden />
                        </span>
                        <span className="min-w-0 flex-1">
                          <span className="line-clamp-2 font-medium text-ink [overflow-wrap:anywhere] group-hover:text-forest">
                            {bank.title}
                          </span>
                          <span className="mt-0.5 block truncate text-sm text-ink-subtle">
                            {[bank.description, t.dashboard.updated(formatRelativeDay(bank.updated_at))]
                              .filter(Boolean)
                              .join(' · ')}
                          </span>
                        </span>
                        <ChevronRight className="size-5 shrink-0 text-ink-subtle" aria-hidden />
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
            </Panel>
          </section>
        </div>

        <aside className="flex flex-col gap-8" aria-label={t.tools.docxTip.title}>
          <DocxTip />
        </aside>
      </div>
    </div>
  )

}

function ContinueBanner({ item }: { item: DashboardQuiz }) {
  const { t, plural } = useI18n()
  const api = useApi()
  const paths = usePaths()
  const attemptId = item.lastAttempt?.attempt_id ?? ''
  const details = useQuery({
    queryKey: queryKeys.attempt(attemptId),
    queryFn: ({ signal }) => api.attempts.get(attemptId, signal),
    enabled: Boolean(attemptId),
  })
  const remaining = useCountdown(details.data?.deadline_at, details.data?.server_time, details.dataUpdatedAt)
  if (remaining === 0) return null

  const total = details.data?.questions.length ?? item.quiz.question_count
  const answered = details.data?.answers.length ?? 0
  const minutesLeft = remaining === null ? null : plural(Math.max(1, Math.ceil(remaining / 60_000)), t.quiz.minutes)
  const progress = total ? (answered / total) * 100 : 0

  return (
    <section
      aria-label={t.dashboard.continueTitle}
      className="mt-8 flex flex-col gap-4 rounded-lg border border-forest/20 bg-forest-tint p-4 sm:flex-row sm:items-center sm:gap-6 sm:p-5"
    >
      <span className="hidden size-12 shrink-0 place-items-center rounded-full bg-surface text-forest ring-1 ring-forest/15 sm:grid">
        <PlayCircle className="size-6" aria-hidden />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-xs font-semibold tracking-wide text-forest uppercase">{t.dashboard.continueTitle}</p>
        <p className="mt-1 line-clamp-2 font-display text-lg leading-snug font-semibold text-ink [overflow-wrap:anywhere]">
          {item.bank?.title ?? t.quiz.untitled}
        </p>
        <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-2">
          <span
            className="h-1.5 w-40 overflow-hidden rounded-full bg-surface ring-1 ring-forest/10"
            role="progressbar"
            aria-valuemin={0}
            aria-valuemax={total}
            aria-valuenow={answered}
            aria-label={t.attempt.progressLabel}
          >
            <span className="block h-full rounded-full bg-forest" style={{ width: `${progress}%` }} />
          </span>
          <span className="text-sm text-ink-muted">
            {details.isPending ? '…' : t.dashboard.continueMeta(t.attempt.answeredOf(answered, total), minutesLeft)}
          </span>
        </div>
      </div>
      <Button asChild size="lg" className={cn('w-full sm:w-auto')}>
        <Link to={paths.attempt(attemptId)}>
          {t.dashboard.continueAction}
          <ArrowRight aria-hidden />
        </Link>
      </Button>
    </section>
  )
}

function DocxTip() {
  const { t } = useI18n()
  return (
    <div className="rounded-lg border border-dashed border-control/50 p-4">
      <p className="flex items-center gap-2 text-sm font-semibold text-ink">
        <Lightbulb className="size-4 text-terracotta" aria-hidden />
        {t.tools.docxTip.title}
      </p>
      <ul className="mt-2 space-y-1.5 text-sm leading-relaxed text-ink-muted">
        {t.tools.docxTip.items.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </div>
  )
}
