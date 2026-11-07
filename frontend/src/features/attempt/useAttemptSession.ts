import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useApi } from '@/api/context'
import { isApiError, isTransient } from '@/api/errors'
import { queryKeys } from '@/api/queryKeys'
import type { AttemptDetails, AttemptResult } from '@/api/types'
import { answersFromServer, nextIndex, resumeIndex, sendWithRetry, tally, type AnswerMap } from './attemptFlow'
import { useCountdown } from './useCountdown'

export type AttemptPhase =
  | 'loading'
  | 'active'
  /** Local timer reached the deadline; inputs are locked. */
  | 'expired'
  /** Server says the attempt is no longer in progress (other tab, 409…). */
  | 'closed'
  | 'finishing'
  | 'finished'

/** The answer being sent right now, or the one whose sending failed. */
export interface PendingAnswer {
  questionId: string
  optionId: string
  status: 'sending' | 'error'
  error?: unknown
}

export type FinishOutcome = { ok: true; result: AttemptResult } | { ok: false; error: unknown }

/**
 * One attempt in "check as you go" mode: every choice is sent at once, is
 * final, and comes back with the right answer. Questions are shown one by one
 * in order; there is no jumping between them.
 */
export function useAttemptSession(attemptId: string) {
  const api = useApi()
  const queryClient = useQueryClient()
  const [closedByServer, setClosedByServer] = useState(false)
  const [finishing, setFinishing] = useState(false)
  const [finished, setFinished] = useState(false)
  const [local, setLocal] = useState<AnswerMap>({})
  const [pending, setPending] = useState<PendingAnswer | null>(null)
  const [cursor, setCursor] = useState<number | null>(null)

  const query = useQuery({
    queryKey: queryKeys.attempt(attemptId),
    queryFn: ({ signal }) => api.attempts.get(attemptId, signal),
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    retry: (count, error) => isTransient(error) && count < 2,
  })
  const attempt: AttemptDetails | undefined = query.data
  const questions = useMemo(() => attempt?.questions ?? [], [attempt])

  // Server answers (after reload or a sync) plus the ones confirmed here.
  const answers = useMemo<AnswerMap>(
    () => ({ ...answersFromServer(attempt?.answers ?? []), ...local }),
    [attempt, local],
  )

  // Pin the student to the first unanswered question once data arrives. The
  // cursor then moves only on "Next", so an answered question keeps showing
  // its verdict until the student moves on.
  if (attempt && cursor === null) setCursor(resumeIndex(questions, answers))
  const index = cursor ?? 0
  const question = questions[index]
  const current = question ? answers[question.id] : undefined
  const upcoming = attempt ? nextIndex(questions, answers, index) : null
  const counts = tally(questions, answers)

  const remaining = useCountdown(attempt?.deadline_at, attempt?.server_time, query.dataUpdatedAt)
  const serverSaysClosed = attempt ? attempt.status !== 'in_progress' : false
  const deadlinePassed = attempt?.deadline_at != null && remaining === 0

  const phase: AttemptPhase = !attempt
    ? 'loading'
    : finished
      ? 'finished'
      : finishing
        ? 'finishing'
        : serverSaysClosed || closedByServer
          ? 'closed'
          : deadlinePassed
            ? 'expired'
            : 'active'

  // Sync with the server whenever the tab becomes visible again.
  const { refetch } = query
  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState === 'visible') void refetch()
    }
    document.addEventListener('visibilitychange', onVisible)
    window.addEventListener('online', onVisible)
    return () => {
      document.removeEventListener('visibilitychange', onVisible)
      window.removeEventListener('online', onVisible)
    }
  }, [refetch])

  // Warn before leaving while an answer is still on its way.
  const sending = pending?.status === 'sending'
  useEffect(() => {
    if (!sending) return
    const onBeforeUnload = (event: BeforeUnloadEvent) => event.preventDefault()
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [sending])

  const busy = useRef(false)
  const send = useCallback(
    async (questionId: string, optionId: string) => {
      if (busy.current) return
      busy.current = true
      setPending({ questionId, optionId, status: 'sending' })
      try {
        const saved = await sendWithRetry(() => api.attempts.saveAnswer(attemptId, questionId, optionId))
        setLocal((prev) => ({
          ...prev,
          [questionId]: {
            optionId: saved.option_id,
            is_correct: saved.is_correct,
            correct_option_ids: saved.correct_option_ids ?? [],
          },
        }))
        setPending(null)
      } catch (error) {
        if (isApiError(error, 'answer_locked')) {
          // Answered in another tab: show what the server has.
          setPending(null)
          void refetch()
        } else if (isApiError(error, 'conflict')) {
          setPending(null)
          setClosedByServer(true)
        } else {
          setPending({ questionId, optionId, status: 'error', error })
        }
      } finally {
        busy.current = false
      }
    },
    [api, attemptId, refetch],
  )

  /** Choose an option: sent at once and final. */
  const choose = useCallback(
    (optionId: string) => {
      if (phase !== 'active' || !question || answers[question.id] || pending) return
      void send(question.id, optionId)
    },
    [answers, pending, phase, question, send],
  )

  /** Re-send the failed choice (the same option: a different one could clash). */
  const retry = useCallback(() => {
    if (pending?.status === 'error' && phase === 'active') void send(pending.questionId, pending.optionId)
  }, [pending, phase, send])

  const goNext = useCallback(() => {
    if (!current || upcoming === null) return false
    setCursor(upcoming)
    return true
  }, [current, upcoming])

  const finishMutation = useMutation({
    mutationFn: () => api.attempts.finish(attemptId),
    retry: false,
  })
  const { mutateAsync: finishAsync } = finishMutation

  const finish = useCallback(async (): Promise<FinishOutcome> => {
    setFinishing(true)
    try {
      const result = await finishAsync()
      queryClient.setQueryData(queryKeys.result(attemptId), result)
      void queryClient.invalidateQueries({ queryKey: ['quizzes', 'history'] })
      void queryClient.invalidateQueries({ queryKey: queryKeys.dashboard })
      setFinished(true)
      return { ok: true, result }
    } catch (error) {
      setFinishing(false)
      if (isApiError(error, 'conflict')) setClosedByServer(true)
      return { ok: false, error }
    }
  }, [attemptId, finishAsync, queryClient])

  return {
    query,
    attempt,
    phase,
    remaining,
    questions,
    index,
    question,
    /** Verdict for the question on screen, once answered. */
    current,
    /** Where "Next" leads; null on the last open question. */
    upcoming,
    counts,
    pending: pending && question && pending.questionId === question.id ? pending : null,
    choose,
    retry,
    goNext,
    finish,
    finishPending: finishMutation.isPending,
    finishError: finishMutation.error,
  }
}

export type AttemptSession = ReturnType<typeof useAttemptSession>
