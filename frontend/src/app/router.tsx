import type { ComponentType } from 'react'
import { createBrowserRouter, type RouteObject } from 'react-router'
import { AppShell } from '@/components/layout/AppShell'
import { FullPageLoader, RequireAuth } from '@/features/auth/RequireAuth'
import { AppRoot } from './AppRoot'

/**
 * Pages are split into their own chunks; the shell, the auth guard and the
 * API layer stay in the entry bundle.
 */
const page = <T,>(load: () => Promise<T>, pick: (module: T) => ComponentType) => ({
  lazy: async () => ({ Component: pick(await load()) }),
})

const appPages: RouteObject[] = [
  { index: true, ...page(() => import('@/features/dashboard/DashboardPage'), (m) => m.DashboardPage) },
  { path: 'banks', ...page(() => import('@/features/banks/BanksPage'), (m) => m.BanksPage) },
  { path: 'banks/:bankId', ...page(() => import('@/features/banks/BankPage'), (m) => m.BankPage) },
  { path: 'import', ...page(() => import('@/features/import/ImportPage'), (m) => m.ImportPage) },
  { path: 'quizzes', ...page(() => import('@/features/quizzes/QuizzesPage'), (m) => m.QuizzesPage) },
  { path: 'quizzes/new', ...page(() => import('@/features/quizzes/NewQuizPage'), (m) => m.NewQuizPage) },
  { path: 'quizzes/:quizId', ...page(() => import('@/features/quizzes/QuizPage'), (m) => m.QuizPage) },
  {
    path: 'attempts/:attemptId/result',
    ...page(() => import('@/features/result/ResultPage'), (m) => m.ResultPage),
  },
]

const attemptPage = page(() => import('@/features/attempt/AttemptPage'), (m) => m.AttemptPage)

const routes: RouteObject[] = [
  {
    element: <AppRoot />,
    hydrateFallbackElement: <FullPageLoader />,
    children: [
      { path: 'login', ...page(() => import('@/features/auth/LoginPage'), (m) => m.LoginPage) },
      { path: 'register', ...page(() => import('@/features/auth/RegisterPage'), (m) => m.RegisterPage) },
      { path: 'check-email', ...page(() => import('@/features/auth/CheckEmailPage'), (m) => m.CheckEmailPage) },
      // Google OAuth callback lands here (backend 303 after setting the session cookie).
      { path: 'auth/complete', ...page(() => import('@/features/auth/AuthCompletePage'), (m) => m.AuthCompletePage) },
      {
        element: <RequireAuth />,
        children: [
          {
            element: <AppShell />,
            children: [
              ...appPages,
              { path: '*', ...page(() => import('./NotFoundPage'), (m) => m.NotFoundPage) },
            ],
          },
          // Focus mode: no app navigation around the attempt.
          { path: 'attempts/:attemptId', ...attemptPage },
        ],
      },
    ],
  },
]

// Phase-1 design preview on an in-memory API. Development builds only.
if (import.meta.env.DEV) {
  routes.unshift({
    path: '/demo',
    hydrateFallbackElement: <FullPageLoader />,
    ...page(() => import('@/demo/DemoLayout'), (m) => m.DemoLayout),
    children: [
      {
        element: <AppShell />,
        children: [
          { index: true, ...page(() => import('@/features/dashboard/DashboardPage'), (m) => m.DashboardPage) },
          {
            path: 'attempts/:attemptId/result',
            ...page(() => import('@/features/result/ResultPage'), (m) => m.ResultPage),
          },
          { path: '*', ...page(() => import('./PlaceholderPage'), (m) => m.PlaceholderPage) },
        ],
      },
      { path: 'attempts/:attemptId', ...attemptPage },
    ],
  })
}

export const router = createBrowserRouter(routes)
