import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router'
import { useApi } from '@/api/context'
import { queryKeys } from '@/api/queryKeys'
import type { UUID } from '@/api/types'
import { usePaths } from '@/app/paths'

export const PAGE_SIZE = 20

export function useQuiz(quizId: UUID | undefined) {
  const api = useApi()
  return useQuery({
    queryKey: queryKeys.quiz(quizId ?? ''),
    queryFn: ({ signal }) => api.quizzes.get(quizId ?? '', signal),
    enabled: Boolean(quizId),
    // A quiz is an immutable snapshot.
    staleTime: Infinity,
  })
}

/** Offset pagination: the last page is shorter than the limit. */
export function useQuizList() {
  const api = useApi()
  return useInfiniteQuery({
    queryKey: [...queryKeys.quizzes({ limit: PAGE_SIZE }), 'infinite'],
    queryFn: ({ pageParam, signal }) => api.quizzes.list({ limit: PAGE_SIZE, offset: pageParam }, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => (last.items.length < last.limit ? undefined : last.offset + last.limit),
  })
}

export function useAttemptHistory(quizId: UUID) {
  const api = useApi()
  return useInfiniteQuery({
    queryKey: [...queryKeys.history(quizId, { limit: PAGE_SIZE }), 'infinite'],
    queryFn: ({ pageParam, signal }) => api.attempts.history(quizId, { limit: PAGE_SIZE, offset: pageParam }, signal),
    initialPageParam: 0,
    getNextPageParam: (last, pages) => (last.length < PAGE_SIZE ? undefined : pages.length * PAGE_SIZE),
  })
}

/**
 * Start (201) or resume (200) the attempt and open it. The response already
 * is the full attempt, so the attempt screen renders without another request.
 * Not retried automatically: the user presses the button again.
 */
export function useStartAttempt(quizId: UUID) {
  const api = useApi()
  const paths = usePaths()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.attempts.start(quizId),
    onSuccess: ({ attempt }) => {
      queryClient.setQueryData(queryKeys.attempt(attempt.id), attempt)
      void queryClient.invalidateQueries({ queryKey: ['quizzes', 'history', quizId] })
      void navigate(paths.attempt(attempt.id))
    },
  })
}
