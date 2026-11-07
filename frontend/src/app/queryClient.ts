import { QueryClient } from '@tanstack/react-query'
import { isTransient } from '@/api/errors'

export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        // Only reads are retried, and only for transient failures.
        retry: (failureCount, error) => isTransient(error) && failureCount < 2,
        refetchOnWindowFocus: true,
      },
      mutations: {
        // Creations and import confirmations must never repeat on their own.
        retry: false,
      },
    },
  })
}
