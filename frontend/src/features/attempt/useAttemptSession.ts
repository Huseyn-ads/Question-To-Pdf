import { useCallback, useEffect, useMemo, useState, useSyncExternalStore } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useApi } from '@/api/context'
import { isApiError, isTransient } from '@/api/errors'
import { queryKeys } from '@/api/queryKeys'
import type { AttemptDetails, AttemptResult } from '@/api/types'
import { AnswerSaver } from './answerSaver'
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

export type FinishOutcome =
  | { ok: true; result: AttemptResult }
  | { ok: false; reason: 'unsaved'; failed: string[] }
  | { ok: false; reason: 'error'; error: unknown }

export function useAttemptSession(attemptId: string) {
  const api = useApi()
  const queryClient = useQueryClient()
  const [closedByServer, setClosedByServer] = useState(false)
  const [finishing, setFinishing] = useState(false)
  const [finished, setFinished] = useState(false)

  const query = useQuery({
    queryKey: queryKeys.attempt(attemptId),
    queryFn: ({ signal }) => api.attempts.get(attemptId, signal),
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    retry: (count, error) => isTransient(error) && count < 2,
  })
  const attempt: AttemptDetails | undefined = query.data
  const receivedAt = query.dataUpdatedAt

  // One saver per attempt for the lifetime of the page (the route remounts per attempt).
  const [saver] = useState(
    () =>
      new AnswerSaver(
        async (questionId, optionId) => {
          await api.attempts.saveAnswer(attemptId, questionId, optionId)
        },
        { onClosed: () => setClosedByServer(true) },
      ),
  )
  const saves = useSyncExternalStore(saver.subscribe, saver.getSnapshot)

  // Seed and re-seed server answers (after reload / tab return).
  useEffect(() => {
    if (!attempt) return
    saver.seed(Object.fromEntries(attempt.answers.map((a) => [a.quiz_question_id, a.quiz_answer_option_id])))
  }, [attempt, saver])

  const remaining = useCountdown(attempt?.deadline_at, attempt?.server_time, receivedAt)

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

  useEffect(() => {
    if (phase === 'closed' || phase === 'expired') saver.stop()
  }, [phase, saver])

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

  // Warn before leaving while a save is still pending.
  useEffect(() => {
    const onBeforeUnload = (event: BeforeUnloadEvent) => {
      if (saver.hasPending()) event.preventDefault()
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [saver])

  const selections = useMemo(() => {
    const map: Record<string, string> = {}
    for (const answer of attempt?.answers ?? []) map[answer.quiz_question_id] = answer.quiz_answer_option_id
    for (const [questionId, state] of Object.entries(saves.questions)) map[questionId] = state.optionId
    return map
  }, [attempt, saves])

  const select = useCallback(
    (questionId: string, optionId: string) => {
      if (phase !== 'active') return
      saver.select(questionId, optionId)
    },
    [phase, saver],
  )

  const finishMutation = useMutation({
    mutationFn: () => api.attempts.finish(attemptId),
    retry: false,
  })
  const { mutateAsync: finishAsync } = finishMutation

  const finish = useCallback(
    async ({ ignoreUnsaved = false }: { ignoreUnsaved?: boolean } = {}): Promise<FinishOutcome> => {
      setFinishing(true)
      try {
        const { failed } = await saver.flush()
        if (failed.length && !ignoreUnsaved) {
          setFinishing(false)
          return { ok: false, reason: 'unsaved', failed }
        }
        saver.stop()
        const result = await finishAsync()
        queryClient.setQueryData(queryKeys.result(attemptId), result)
        void queryClient.invalidateQueries({ queryKey: ['quizzes', 'history'] })
        void queryClient.invalidateQueries({ queryKey: queryKeys.dashboard })
        setFinished(true)
        return { ok: true, result }
      } catch (error) {
        setFinishing(false)
        if (isApiError(error, 'conflict')) setClosedByServer(true)
        return { ok: false, reason: 'error', error }
      }
    },
    [attemptId, finishAsync, queryClient, saver],
  )

  const questionIndexById = useMemo(() => {
    const map = new Map<string, number>()
    attempt?.questions.forEach((q, index) => map.set(q.id, index))
    return map
  }, [attempt])

  return {
    query,
    attempt,
    phase,
    remaining,
    selections,
    saves,
    select,
    retryFailed: () => saver.retryFailed(),
    finish,
    finishError: finishMutation.error,
    questionIndexById,
  }
}

export type AttemptSession = ReturnType<typeof useAttemptSession>
