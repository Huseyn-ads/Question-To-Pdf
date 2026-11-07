import { createContext, useContext, useMemo, type ReactNode } from 'react'

/**
 * Route builders. The design preview lives under /demo, so every link is
 * built relative to a base prefix instead of being hard-coded.
 */
const BaseContext = createContext('')

export function PathsProvider({ base, children }: { base: string; children: ReactNode }) {
  return <BaseContext.Provider value={base}>{children}</BaseContext.Provider>
}

export function buildPaths(base: string) {
  return {
    base,
    dashboard: base || '/',
    banks: `${base}/banks`,
    bank: (bankId: string) => `${base}/banks/${bankId}`,
    importDocx: `${base}/import`,
    importInto: (bankId: string) => `${base}/import?bank=${encodeURIComponent(bankId)}`,
    quizzes: `${base}/quizzes`,
    quiz: (quizId: string) => `${base}/quizzes/${quizId}`,
    newQuiz: `${base}/quizzes/new`,
    newQuizFrom: (bankId: string) => `${base}/quizzes/new?bank=${encodeURIComponent(bankId)}`,
    attempt: (attemptId: string, question?: number) =>
      `${base}/attempts/${attemptId}${question ? `?q=${question}` : ''}`,
    result: (attemptId: string) => `${base}/attempts/${attemptId}/result`,
    login: '/login',
    register: '/register',
  }
}

export type Paths = ReturnType<typeof buildPaths>

export function usePaths(): Paths {
  const base = useContext(BaseContext)
  return useMemo(() => buildPaths(base), [base])
}
