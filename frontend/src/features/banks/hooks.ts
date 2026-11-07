import { useQuery } from '@tanstack/react-query'
import { useApi } from '@/api/context'
import { queryKeys } from '@/api/queryKeys'
import type { UUID } from '@/api/types'

export function useBanks() {
  const api = useApi()
  return useQuery({ queryKey: queryKeys.banks, queryFn: ({ signal }) => api.banks.list(signal) })
}

export function useBank(bankId: UUID | undefined) {
  const api = useApi()
  return useQuery({
    queryKey: queryKeys.bank(bankId ?? ''),
    queryFn: ({ signal }) => api.banks.get(bankId ?? '', signal),
    enabled: Boolean(bankId),
  })
}

export function useBankQuestions(bankId: UUID | undefined) {
  const api = useApi()
  return useQuery({
    queryKey: queryKeys.questions(bankId ?? ''),
    queryFn: ({ signal }) => api.questions.list(bankId ?? '', signal),
    enabled: Boolean(bankId),
  })
}
