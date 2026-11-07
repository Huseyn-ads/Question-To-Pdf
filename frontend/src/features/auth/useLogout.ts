import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router'
import { useApi } from '@/api/context'
import { isApiError } from '@/api/errors'
import { usePaths } from '@/app/paths'
import { useT } from '@/i18n'
import { useToast } from '@/components/common/toast'
import { endSession } from './session'

/**
 * POST /auth/logout, then forget the profile, CSRF token and every cached
 * resource. A 401 means the session is already gone, which is the same outcome.
 */
export function useLogout() {
  const api = useApi()
  const t = useT()
  const toast = useToast()
  const paths = usePaths()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      try {
        await api.auth.logout()
      } catch (error) {
        if (!isApiError(error, 'unauthorized')) throw error
      }
    },
    onSuccess: () => {
      endSession(queryClient, 'logout')
      // The demo has no route guard, so leave it explicitly.
      if (paths.base) void navigate(paths.login, { replace: true })
    },
    onError: () => toast.error(t.auth.logoutFailed),
  })
}
