import { useQueries, useQuery } from '@tanstack/react-query'
import { useApi } from '@/api/context'
import { queryKeys } from '@/api/queryKeys'
import type { QuizRowItem } from '@/features/quizzes/QuizRow'
import { useMe } from '@/features/auth/session'

export const RECENT_QUIZZES = 5

export type DashboardQuiz = QuizRowItem

/**
 * The backend has no "my active attempts" endpoint, so the dashboard reads
 * the latest attempt of each recent quiz (limit=1). Quiz titles come from the
 * bank list because quizzes have no title of their own.
 */
export function useDashboardData() {
  const api = useApi()
  const me = useMe()
  const banks = useQuery({ queryKey: queryKeys.banks, queryFn: ({ signal }) => api.banks.list(signal) })
  const quizzes = useQuery({
    queryKey: queryKeys.quizzes({ limit: RECENT_QUIZZES }),
    queryFn: ({ signal }) => api.quizzes.list({ limit: RECENT_QUIZZES }, signal),
  })
  const histories = useQueries({
    queries: (quizzes.data?.items ?? []).map((quiz) => ({
      queryKey: queryKeys.history(quiz.id, { limit: 1 }),
      queryFn: ({ signal }: { signal: AbortSignal }) => api.attempts.history(quiz.id, { limit: 1 }, signal),
    })),
  })

  const bankById = new Map((banks.data ?? []).map((bank) => [bank.id, bank]))
  const recent: DashboardQuiz[] = (quizzes.data?.items ?? []).map((quiz, index) => {
    const history = histories[index]
    return {
      quiz,
      bank: bankById.get(quiz.question_bank_id),
      lastAttempt: history?.data ? (history.data[0] ?? null) : undefined,
    }
  })
  const active = recent.find((item) => item.lastAttempt?.status === 'in_progress')

  return {
    me,
    banks,
    quizzes,
    recent,
    active,
    historiesLoading: histories.some((h) => h.isPending),
  }
}
