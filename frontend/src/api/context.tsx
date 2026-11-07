import { createContext, useContext, type ReactNode } from 'react'
import { api as realApi, type Api } from './endpoints'

/**
 * Pages read the API through context so the phase-1 design preview can swap
 * in an in-memory demo implementation with the same interface. Production
 * routes always use the real backend client.
 */
const ApiContext = createContext<Api>(realApi)

export function ApiProvider({ api, children }: { api: Api; children: ReactNode }) {
  return <ApiContext.Provider value={api}>{children}</ApiContext.Provider>
}

export function useApi(): Api {
  return useContext(ApiContext)
}
