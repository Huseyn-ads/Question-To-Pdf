import { useEffect, useState } from 'react'
import { Outlet, ScrollRestoration } from 'react-router'
import { QueryClientProvider } from '@tanstack/react-query'
import { setUnauthorizedListener } from '@/api/http'
import { queryKeys } from '@/api/queryKeys'
import { endSession } from '@/features/auth/session'
import { createQueryClient } from './queryClient'

/**
 * Root of the real application: one query cache for the signed-in user and
 * the global 401 hook. A 401 on any protected request ends the local session;
 * the route guard then sends the user to /login with a return URL.
 */
export function AppRoot() {
  const [queryClient] = useState(createQueryClient)

  useEffect(() => {
    setUnauthorizedListener(() => {
      const hadSession = Boolean(queryClient.getQueryData(queryKeys.me))
      endSession(queryClient, hadSession ? 'expired' : undefined)
    })
    return () => setUnauthorizedListener(null)
  }, [queryClient])

  return (
    <QueryClientProvider client={queryClient}>
      <ScrollRestoration />
      <Outlet />
    </QueryClientProvider>
  )
}
