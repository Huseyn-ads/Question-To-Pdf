import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, ArrowLeft, ArrowRight, Flag, Grid3x3, TimerOff, X } from 'lucide-react'
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
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { ErrorState } from '@/components/common/states'
import { LogoMark } from '@/components/brand/Logo'
import { FinishDialog } from './FinishDialog'
import { QuestionNavigator } from './QuestionNavigator'
import { QuestionView } from './QuestionView'
import { SaveStatus } from './SaveStatus'
import { TimerPill } from './TimerPill'
import { useAttemptSession } from './useAttemptSession'

export function AttemptPage() {
  const { attemptId = '' } = useParams()
  // Remount per attempt so the save queue never leaks between attempts.
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
  const { attempt, phase } = session
  const title = useQuizTitle(attempt?.quiz_id)
  const [searchParams, setSearchParams] = useSearchParams()
  const [finishOpen, setFinishOpen] = useState(false)
  const [sheetOpen, setSheetOpen] = useState(false)
  const headingRef = useRef<HTMLHeadingElement>(null)

  const questions = useMemo(() => attempt?.questions ?? [], [attempt])
  const total = questions.length
  const requested = Number.parseInt(searchParams.get('q') ?? '1', 10)
  const index = Math.min(Math.max(Number.isFinite(requested) ? requested - 1 : 0, 0), Math.max(total - 1, 0))
  const question = questions[index]
  const answeredCount = questions.filter((q) => session.selections[q.id]).length
  const unansweredNumbers = questions.flatMap((q, i) => (session.selections[q.id] ? [] : [i + 1]))
  const locked = phase !== 'active'

  // The question the user is on right now. Updated synchronously on every move,
  // so a key pressed before the next render never lands on the previous question.
  const currentIndex = useRef(index)
  useEffect(() => {
    currentIndex.current = index
  }, [index])

  const goTo = useCallback(
    (next: number, { focus = true } = {}) => {
      if (next < 0 || next >= total) return
      currentIndex.current = next
      setSearchParams({ q: String(next + 1) }, { replace: true })
      window.scrollTo({ top: 0, behavior: 'instant' })
      if (focus) requestAnimationFrame(() => headingRef.current?.focus({ preventScroll: true }))
    },
    [setSearchParams, total],
  )

  const toResult = useCallback(() => void navigate(paths.result(attemptId), { replace: true }), [navigate, paths, attemptId])

  // Keyboard: 1–9 choose an option; ←/→ change question (radios keep their own arrows).
  const dialogOpen = finishOpen || sheetOpen || phase === 'expired' || phase === 'closed'
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (dialogOpen || event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey) return
      const target = event.target as HTMLElement | null
      const isRadio = target instanceof HTMLInputElement && target.type === 'radio'
      if (target?.closest('input:not([type=radio]), textarea, select, [contenteditable=true]')) return
      const at = currentIndex.current
      const current = questions[at]
      if (current && /^[1-9]$/.test(event.key)) {
        const option = current.options[Number(event.key) - 1]
        if (option && !locked) {
          event.preventDefault()
          session.select(current.id, option.id)
        }
      } else if (!isRadio && event.key === 'ArrowRight') {
        event.preventDefault()
        goTo(at + 1)
      } else if (!isRadio && event.key === 'ArrowLeft') {
        event.preventDefault()
        goTo(at - 1)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [dialogOpen, goTo, locked, questions, session])

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
  const saveStatus = <SaveStatus overall={session.saves.overall} onRetry={session.retryFailed} />
  const isLast = index === total - 1

  return (
    <FocusFrame>
      {/* Header */}
      <header className="sticky top-0 z-20 border-b border-line bg-surface/95 backdrop-blur-sm">
        <div className="mx-auto flex h-16 max-w-[76rem] items-center gap-3 px-3 sm:px-6">
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
              <p className="truncate text-xs text-ink-subtle">
                {t.attempt.answeredOf(answeredCount, total)}
              </p>
            )}
          </div>
          {attempt && <TimerPill remaining={session.remaining} compact />}
        </div>
        <div
          className="h-[3px] bg-sunken"
          role="progressbar"
          aria-label={t.attempt.progressLabel}
          aria-valuemin={0}
          aria-valuemax={total}
          aria-valuenow={answeredCount}
          aria-valuetext={t.attempt.answeredOf(answeredCount, total)}
        >
          <div
            className="h-full bg-forest transition-[width] duration-300"
            style={{ width: total ? `${(answeredCount / total) * 100}%` : 0 }}
          />
        </div>
      </header>

      <div className="mx-auto grid max-w-[76rem] gap-10 px-4 pt-6 pb-32 sm:px-6 md:pt-10 lg:grid-cols-[minmax(0,1fr)_17.5rem] lg:pb-16 xl:gap-14">
        <main id="main" className="min-w-0">
          <div className="mx-auto max-w-reading">
            {!attempt || !question ? (
              <QuestionSkeleton />
            ) : (
              <>
                {session.saves.overall === 'error' && phase === 'active' && (
                  <Alert tone="danger" className="mb-6">
                    <AlertTriangle aria-hidden />
                    <AlertTitle>{t.attempt.save.error}</AlertTitle>
                    <AlertDescription>
                      <p>{t.attempt.save.errorDetail}</p>
                      <Button size="sm" variant="secondary" onClick={session.retryFailed}>
                        {t.attempt.save.retry}
                      </Button>
                    </AlertDescription>
                  </Alert>
                )}
                <QuestionView
                  ref={headingRef}
                  question={question}
                  number={index + 1}
                  total={total}
                  selectedOptionId={session.selections[question.id]}
                  disabled={locked}
                  onSelect={(optionId) => session.select(question.id, optionId)}
                  status={saveStatus}
                />

                {/* Desktop / tablet navigation */}
                <div className="mt-8 hidden items-center justify-between gap-3 border-t border-line-soft pt-6 sm:flex">
                  <Button variant="secondary" onClick={() => goTo(index - 1)} disabled={index === 0}>
                    <ArrowLeft aria-hidden />
                    {t.attempt.prev}
                  </Button>
                  <p className="hidden text-xs text-ink-subtle lg:block">{t.attempt.keyboardHint}</p>
                  <Button variant="ghost" className="lg:hidden" onClick={() => setSheetOpen(true)}>
                    <Grid3x3 aria-hidden />
                    {t.attempt.allQuestions}
                    <span className="tabular text-ink-subtle">
                      {index + 1}/{total}
                    </span>
                  </Button>
                  {isLast ? (
                    <Button onClick={() => setFinishOpen(true)} disabled={locked}>
                      <Flag aria-hidden />
                      {t.attempt.toFinish}
                    </Button>
                  ) : (
                    <Button onClick={() => goTo(index + 1)}>
                      {t.attempt.next}
                      <ArrowRight aria-hidden />
                    </Button>
                  )}
                </div>
              </>
            )}
          </div>
        </main>

        {/* Desktop navigator */}
        <aside className="hidden lg:block" aria-label={t.attempt.navigatorTitle}>
          <div className="sticky top-24 rounded-lg border border-line bg-surface p-5">
            <div className="mb-4 flex items-baseline justify-between gap-2">
              <h2 className="font-display text-lg font-semibold text-ink">{t.attempt.navigatorTitle}</h2>
              <span className="tabular text-sm text-ink-muted">
                {answeredCount}/{total}
              </span>
            </div>
            {attempt ? (
              <QuestionNavigator
                questions={questions}
                currentIndex={index}
                selections={session.selections}
                saves={session.saves}
                onJump={(i) => goTo(i)}
              />
            ) : (
              <div className="grid grid-cols-5 gap-2">
                {Array.from({ length: 10 }, (_, i) => (
                  <Skeleton key={i} className="h-10" />
                ))}
              </div>
            )}
            <Button className="mt-5 w-full" onClick={() => setFinishOpen(true)} disabled={!attempt || locked}>
              <Flag aria-hidden />
              {t.attempt.finish}
            </Button>
          </div>
        </aside>
      </div>

      {/* Mobile bottom bar */}
      {attempt && (
        <div className="pb-safe fixed inset-x-0 bottom-0 z-20 border-t border-line bg-surface/97 px-3 pt-3 backdrop-blur-sm sm:hidden">
          <div className="grid grid-cols-[auto_1fr_auto] items-center gap-2">
            <Button
              variant="secondary"
              size="icon"
              onClick={() => goTo(index - 1)}
              disabled={index === 0}
              aria-label={t.attempt.prev}
            >
              <ArrowLeft aria-hidden />
            </Button>
            <Button variant="ghost" onClick={() => setSheetOpen(true)} className="text-ink">
              <Grid3x3 aria-hidden />
              <span className="tabular">
                {index + 1} / {total}
              </span>
              <span className="sr-only">{t.attempt.allQuestions}</span>
            </Button>
            {isLast ? (
              <Button onClick={() => setFinishOpen(true)} disabled={locked} className="px-4">
                <Flag aria-hidden />
                {t.attempt.toFinish}
              </Button>
            ) : (
              <Button onClick={() => goTo(index + 1)} className="px-5">
                {t.attempt.next}
                <ArrowRight aria-hidden />
              </Button>
            )}
          </div>
        </div>
      )}

      <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
        <SheetContent side="bottom" closeLabel={t.common.close} className="px-4 pb-6">
          <SheetHeader className="px-0 pb-0">
            <SheetTitle>{t.attempt.allQuestions}</SheetTitle>
            <SheetDescription>{t.attempt.answeredOf(answeredCount, total)}</SheetDescription>
          </SheetHeader>
          <div className="overflow-y-auto">
            <QuestionNavigator
              questions={questions}
              currentIndex={index}
              selections={session.selections}
              saves={session.saves}
              columns={6}
              onJump={(i) => {
                setSheetOpen(false)
                goTo(i, { focus: false })
              }}
            />
          </div>
          <Button
            size="lg"
            className="w-full"
            disabled={locked}
            onClick={() => {
              setSheetOpen(false)
              setFinishOpen(true)
            }}
          >
            <Flag aria-hidden />
            {t.attempt.finish}
          </Button>
        </SheetContent>
      </Sheet>

      <FinishDialog
        open={finishOpen && (phase === 'active' || phase === 'finishing')}
        onOpenChange={setFinishOpen}
        unansweredNumbers={unansweredNumbers}
        total={total}
        numberOf={(id) => (session.questionIndexById.get(id) ?? 0) + 1}
        onJump={(n) => {
          setFinishOpen(false)
          goTo(n - 1)
        }}
        onFinish={session.finish}
        onRetrySaves={session.retryFailed}
        onFinished={toResult}
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
          <span className={cn('mb-1 grid size-11 place-items-center rounded-full bg-terracotta-soft text-[#8a4527]')}>
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
