import type { PageParams, UUID } from './types'

export const queryKeys = {
  me: ['auth', 'me'] as const,
  /** Why the last session ended locally ('logout' | 'expired'); never fetched. */
  sessionEnd: ['auth', 'session-end'] as const,
  banks: ['banks'] as const,
  bank: (bankId: UUID) => ['banks', bankId] as const,
  questions: (bankId: UUID) => ['banks', bankId, 'questions'] as const,
  quizzes: (params: PageParams = {}) => ['quizzes', 'list', params] as const,
  quiz: (quizId: UUID) => ['quizzes', 'detail', quizId] as const,
  history: (quizId: UUID, params: PageParams = {}) => ['quizzes', 'history', quizId, params] as const,
  attempt: (attemptId: UUID) => ['attempts', attemptId] as const,
  result: (attemptId: UUID) => ['attempts', attemptId, 'result'] as const,
  dashboard: ['dashboard'] as const,
}
