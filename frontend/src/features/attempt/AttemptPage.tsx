import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, ArrowLeft, ArrowRight, CheckCircle2, Flag, TimerOff, X, XCircle } from 'lucide-react'
import { useApi } from '@/api/context'
import { isApiError } from '@/api/errors'
import { queryKeys } from '@/api/queryKeys'
import { usePaths } from '@/app/paths'
import { useI18n } from '@/i18n'
import { cn } from '@/lib/utils'
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
import { Skeleton } from '@/components/ui/skeleton'
import { ErrorState, errorMessage } from '@/components/common/states'
import { LogoMark } from '@/components/brand/Logo'
import { optionLetter } from './optionLetter'
import { QuestionView } from './QuestionView'
import { TimerPill } from './TimerPill'
import { useAttemptSession, type AttemptSession } from './useAttemptSession'

export function AttemptPage() {
  const { attemptId = '' } = useParams()
  // Remount per attempt so local answers never leak between attempts.
  return <AttemptScreen key={attemptId} attemptId={attemptId} />
}

function useQuizTitle(quizId: string | undefined) {
  const api = useApi()
  const quiz = useQuery({
    queryKey: queryKeys.quiz(quizId ?? ''),
    queryFn: ({ signal }) => api.quizzes.get(quizId ?? '', signal),
    enabled: Boolean(quizId),
    staleTime: Infinity,
  })
  const bankId = quiz.data?.question_bank_id
  const bank = useQuery({
    queryKey: queryKeys.bank(bankId ?? ''),
    queryFn: ({ signal }) => api.banks.get(bankId ?? '', signal),
    enabled: Boolean(bankId),
  })
  return bank.data?.title
}

function AttemptScreen({ attemptId }: { attemptId: string }) {
  const { t } = useI18n()
  const paths = usePaths()
  const navigate = useNavigate()
  const session = useAttemptSession(attemptId)
  const { attempt, phase, question, current, index, questions, counts } = session
  const title = useQuizTitle(attempt?.quiz_id)
  const [earlyOpen, setEarlyOpen] = useState(false)
  const [finishError, setFinishError] = useState<unknown>(null)
  const headingRef = useRef<HTMLHeadingElement>(null)
  const nextRef = useRef<HTMLButtonElement>(null)

  const total = questions.length
  const locked = phase !== 'active'
  const isDone = Boolean(current) && session.upcoming === null

  const toResult = useCallback(() => void navigate(paths.result(attemptId), { replace: true }), [navigate, paths, attemptId])

  const finish = useCallback(async () => {
    setFinishError(null)
    const outcome = await session.finish()
    if (outcome.ok) toResult()
    else setFinishError(outcome.error)
    return outcome.ok
  }, [session, toResult])

  const goNext = useCallback(() => {
    if (!session.goNext()) return
    window.scrollTo({ top: 0, behavior: 'instant' })
    requestAnimationFrame(() => headingRef.current?.focus({ preventScroll: true }))
  }, [session])

  // After the verdict, move focus to "Next" so Enter continues.
  const answeredId = current ? question?.id : undefined
  useEffect(() => {
    if (answeredId) requestAnimationFrame(() => nextRef.current?.focus({ preventScroll: true }))
  }, [answeredId])

  // Keyboard: 1–9 choose an option; Enter or → goes on after the verdict.
  const dialogOpen = earlyOpen || phase === 'expired' || phase === 'closed'
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (dialogOpen || event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey) return
      const target = event.target as HTMLElement | null
      if (target?.closest('input, textarea, select, [contenteditable=true]')) return
      if (!question) return
      if (!current && /^[1-9]$/.test(event.key)) {
        const option = question.options[Number(event.key) - 1]
        if (option && !locked) {
          event.preventDefault()
          session.choose(option.id)
        }
      } else if (current && event.key === 'ArrowRight' && session.upcoming !== null) {
        event.preventDefault()
        goNext()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [current, dialogOpen, goNext, locked, question, session])

  if (session.query.isError && !attempt) {
    const notFound = isApiError(session.query.error, 'not_found') || isApiError(session.query.error, 'validation')
    return (
      <FocusFrame>
        <div className="mx-auto max-w-lg px-4 py-16">
          <ErrorState
            title={t.attempt.loadError}
            error={session.query.error}
            onRetry={notFound ? undefined : () => void session.query.refetch()}
            retrying={session.query.isFetching}
          />
          <div className="mt-2 flex justify-center">
            <Button asChild variant="link">
              <Link to={paths.dashboard}>{t.result.toDashboard}</Link>
            </Button>
          </div>
        </div>
      </FocusFrame>
    )
  }

  const exitTo = attempt ? paths.quiz(attempt.quiz_id) : paths.dashboard

  return (
    <FocusFrame>
      <header className="sticky top-0 z-20 border-b border-line bg-surface/95 backdrop-blur-sm">
        <div className="mx-auto flex h-16 max-w-[60rem] items-center gap-2 px-3 sm:gap-3 sm:px-6">
          <Button asChild variant="ghost" size="sm" className="hidden sm:inline-flex">
            <Link to={exitTo} title={t.attempt.exitHint}>
              <ArrowLeft aria-hidden />
              {t.attempt.exit}
            </Link>
          </Button>
          <Button asChild variant="ghost" size="icon-sm" className="sm:hidden">
            <Link to={exitTo} aria-label={`${t.attempt.exit}. ${t.attempt.exitHint}`}>
              <X aria-hidden />
            </Link>
          </Button>
          <span className="hidden h-6 w-px bg-line sm:block" aria-hidden />
          <LogoMark className="hidden size-7 md:block" />
          <div className="min-w-0 flex-1">
            <p className="truncate text-[0.9375rem] font-medium text-ink">
              {title ?? (attempt ? t.quiz.untitled : <Skeleton className="h-4 w-40" />)}
            </p>
            {attempt && (
              <p className="truncate text-xs text-ink-subtle">{t.attempt.questionOf(Math.min(index + 1, total), total)}</p>
            )}
          </div>
          {attempt && <Score correct={counts.correct} wrong={counts.wrong} />}
          {attempt && <TimerPill remaining={session.remaining} compact />}
        </div>
        <div
          className="h-[3px] bg-sunken"
          role="progressbar"
          aria-label={t.attempt.progressLabel}
          aria-valuemin={0}
          aria-valuemax={total}
          aria-valuenow={counts.answered}
          aria-valuetext={t.attempt.answeredOf(counts.answered, total)}
        >
          <div
            className="h-full bg-forest transition-[width] duration-300"
            style={{ width: total ? `${(counts.answered / total) * 100}%` : 0 }}
          />
        </div>
      </header>

      <main id="main" className="mx-auto max-w-[60rem] px-4 pt-6 pb-36 sm:px-6 md:pt-10 md:pb-16">
        <div className="mx-auto max-w-reading">
          {!attempt || !question ? (
            <QuestionSkeleton />
          ) : (
            <>
              <QuestionView
                key={question.id}
                ref={headingRef}
                question={question}
                number={index + 1}
                total={total}
                answer={current}
                sendingId={session.pending?.status === 'sending' ? session.pending.optionId : undefined}
                disabled={locked}
                onChoose={session.choose}
              />

              <div aria-live="polite" className="mt-5 empty:hidden">
                {current && <Verdict session={session} />}
                {session.pending?.status === 'error' && (
                  <Alert tone="danger">
                    <AlertTriangle aria-hidden />
                    <AlertTitle>{t.attempt.sendFailed}</AlertTitle>
                    <AlertDescription>
                      <p>{errorMessage(t, session.pending.error)}</p>
                      <Button size="sm" variant="secondary" onClick={session.retry}>
                        {t.common.retry}
                      </Button>
                    </AlertDescription>
                  </Alert>
                )}
              </div>

              {finishError != null && (
                <Alert tone="danger" className="mt-5">
                  <AlertTriangle aria-hidden />
                  <AlertTitle>{t.attempt.finishFailed}</AlertTitle>
                  <AlertDescription>
                    <p>{errorMessage(t, finishError)}</p>
                  </AlertDescription>
                </Alert>
              )}

              {/* Actions: inline on tablets and desktops, pinned to the bottom on phones. */}
              <div className="pb-safe fixed inset-x-0 bottom-0 z-20 border-t border-line bg-surface/97 px-4 pt-3 backdrop-blur-sm sm:static sm:mt-8 sm:border-t sm:border-line-soft sm:bg-transparent sm:px-0 sm:pt-6 sm:pb-0 sm:backdrop-blur-none">
                <div className="flex items-center justify-between gap-3">
                  {!isDone ? (
                    <Button
                      variant="ghost"
                      className="text-ink-muted"
                      onClick={() => setEarlyOpen(true)}
                      disabled={locked || session.pending?.status === 'sending'}
                    >
                      <Flag aria-hidden />
                      <span className="sm:hidden">{t.attempt.finishShort}</span>
                      <span className="hidden sm:inline">{t.attempt.finishEarly}</span>
                    </Button>
                  ) : (
                    <span className="hidden text-sm text-ink-muted sm:block">{t.attempt.allDone}</span>
                  )}
                  {isDone ? (
                    <Button
                      ref={nextRef}
                      size="lg"
                      className="ml-auto max-sm:flex-1"
                      loading={session.finishPending}
                      disabled={locked && phase !== 'finishing'}
                      onClick={() => void finish()}
                    >
                      {t.attempt.showResult}
                      <ArrowRight aria-hidden />
                    </Button>
                  ) : (
                    <Button
                      ref={nextRef}
                      size="lg"
                      className="ml-auto max-sm:flex-1"
                      onClick={goNext}
                      disabled={!current || locked}
                    >
                      {t.attempt.next}
                      <ArrowRight aria-hidden />
                    </Button>
                  )}
                </div>
                <p className="mt-3 hidden text-center text-xs text-ink-subtle lg:block">
                  {current ? t.attempt.keyboardNext : t.attempt.keyboardHint}
                </p>
              </div>
            </>
          )}
        </div>
      </main>

      <EarlyFinishDialog
        open={earlyOpen && (phase === 'active' || phase === 'finishing')}
        onOpenChange={setEarlyOpen}
        unanswered={counts.unanswered}
        pending={session.finishPending}
        error={finishError}
        onConfirm={() => void finish()}
      />

      <BlockingDialog
        open={phase === 'expired'}
        icon={<TimerOff className="size-5" aria-hidden />}
        title={t.attempt.expiredDialog.title}
        text={t.attempt.expiredDialog.text}
        action={t.attempt.expiredDialog.action}
        onAction={toResult}
      />
      <BlockingDialog
        open={phase === 'closed'}
        icon={<Flag className="size-5" aria-hidden />}
        title={t.attempt.closedDialog.title}
        text={t.attempt.closedDialog.text}
        action={t.attempt.closedDialog.action}
        onAction={toResult}
      />
    </FocusFrame>
  )
}

/** Running score: answers are checked at once, so the student sees it grow. */
function Score({ correct, wrong }: { correct: number; wrong: number }) {
  const { t } = useI18n()
  return (
    <p className="flex items-center gap-2 text-sm font-semibold tabular" aria-label={t.attempt.scoreLabel(correct, wrong)}>
      <span className="inline-flex items-center gap-1 rounded-full bg-forest-soft px-2 py-1 text-forest" aria-hidden>
        <CheckCircle2 className="size-3.5" />
        {correct}
      </span>
      <span className="inline-flex items-center gap-1 rounded-full bg-terracotta-soft px-2 py-1 text-terracotta-ink" aria-hidden>
        <XCircle className="size-3.5" />
        {wrong}
      </span>
    </p>
  )
}

function Verdict({ session }: { session: AttemptSession }) {
  const { t } = useI18n()
  const { current, question } = session
  if (!current || !question) return null
  const noKey = current.correct_option_ids.length === 0
  const right = question.options.flatMap((o, i) =>
    current.correct_option_ids.includes(o.id) ? [`${optionLetter(i)} — ${o.text}`] : [],
  )

  if (noKey) {
    return (
      <Alert tone="warning">
        <AlertTriangle aria-hidden />
        <AlertTitle>{t.attempt.feedback.noKeyTitle}</AlertTitle>
        <AlertDescription>
          <p>{t.attempt.feedback.noKey}</p>
        </AlertDescription>
      </Alert>
    )
  }

  return (
    <div
      className={cn(
        'flex gap-3 rounded-lg border px-4 py-3.5 animate-fade-in',
        current.is_correct ? 'border-forest/25 bg-forest-tint' : 'border-terracotta/35 bg-terracotta-soft',
      )}
    >
      {current.is_correct ? (
        <CheckCircle2 className="mt-0.5 size-5 shrink-0 text-forest" aria-hidden />
      ) : (
        <XCircle className="mt-0.5 size-5 shrink-0 text-terracotta-ink" aria-hidden />
      )}
      <div className="min-w-0">
        <p className={cn('font-semibold', current.is_correct ? 'text-forest' : 'text-terracotta-ink')}>
          {current.is_correct ? t.attempt.feedback.rightTitle : t.attempt.feedback.wrongTitle}
        </p>
        {!current.is_correct && (
          <p className="mt-0.5 text-sm whitespace-pre-wrap text-ink [overflow-wrap:anywhere]">
            {t.attempt.feedback.correctIs(right.join('; '))}
          </p>
        )}
      </div>
    </div>
  )
}

function EarlyFinishDialog({
  open,
  onOpenChange,
  unanswered,
  pending,
  error,
  onConfirm,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  unanswered: number
  pending: boolean
  error: unknown
  onConfirm: () => void
}) {
  const { t, plural } = useI18n()
  return (
    <Dialog open={open} onOpenChange={(next) => !pending && onOpenChange(next)}>
      <DialogContent showCloseButton={!pending}>
        <DialogHeader>
          <DialogTitle>{t.attempt.earlyDialog.title}</DialogTitle>
          <DialogDescription>{t.attempt.earlyDialog.text(plural(unanswered, t.quiz.questions))}</DialogDescription>
        </DialogHeader>
        {error != null && (
          <Alert tone="danger">
            <AlertTriangle aria-hidden />
            <AlertDescription>
              <p>{errorMessage(t, error)}</p>
            </AlertDescription>
          </Alert>
        )}
        <DialogFooter>
          <Button variant="secondary" disabled={pending} onClick={() => onOpenChange(false)}>
            {t.attempt.earlyDialog.keepGoing}
          </Button>
          <Button loading={pending} onClick={onConfirm}>
            {error != null ? t.common.retry : t.attempt.earlyDialog.confirm}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** Distraction-free frame: no app navigation while answering. */
function FocusFrame({ children }: { children: ReactNode }) {
  return <div className="min-h-dvh bg-paper">{children}</div>
}

function QuestionSkeleton() {
  return (
    <div aria-busy="true" className="space-y-4">
      <Skeleton className="h-4 w-28" />
      <Skeleton className="h-7 w-full" />
      <Skeleton className="h-7 w-4/5" />
      <div className="space-y-2.5 pt-5">
        {Array.from({ length: 4 }, (_, i) => (
          <Skeleton key={i} className="h-14 w-full" />
        ))}
      </div>
    </div>
  )
}

function BlockingDialog({
  open,
  icon,
  title,
  text,
  action,
  onAction,
}: {
  open: boolean
  icon: ReactNode
  title: string
  text: string
  action: string
  onAction: () => void
}) {
  return (
    <Dialog open={open}>
      <DialogContent
        showCloseButton={false}
        onEscapeKeyDown={(e) => e.preventDefault()}
        onPointerDownOutside={(e) => e.preventDefault()}
        onInteractOutside={(e) => e.preventDefault()}
      >
        <DialogHeader className="pr-0">
          <span className="mb-1 grid size-11 place-items-center rounded-full bg-terracotta-soft text-terracotta-ink">
            {icon}
          </span>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{text}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button size="lg" onClick={onAction} className="w-full sm:w-auto">
            {action}
            <ArrowRight aria-hidden />
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
