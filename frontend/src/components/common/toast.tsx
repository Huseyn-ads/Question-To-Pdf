import { createContext, useCallback, useContext, useMemo, useRef, useState, type ReactNode } from 'react'
import { AlertCircle, CheckCircle2, X } from 'lucide-react'
import { useT } from '@/i18n'
import { cn } from '@/lib/utils'

type ToastTone = 'success' | 'danger'
interface ToastItem {
  id: number
  tone: ToastTone
  text: string
}

interface ToastApi {
  success: (text: string) => void
  error: (text: string) => void
}

const ToastContext = createContext<ToastApi | null>(null)

const DURATION = 5000

/**
 * Small confirmations after an action ("Банк создан"). Announced through a
 * polite live region; errors that need a decision stay inline instead.
 */
export function ToastProvider({ children }: { children: ReactNode }) {
  const t = useT()
  const [items, setItems] = useState<ToastItem[]>([])
  const nextId = useRef(1)

  const dismiss = useCallback((id: number) => setItems((list) => list.filter((item) => item.id !== id)), [])
  const push = useCallback(
    (tone: ToastTone, text: string) => {
      const id = nextId.current++
      setItems((list) => [...list.slice(-2), { id, tone, text }])
      window.setTimeout(() => dismiss(id), DURATION)
    },
    [dismiss],
  )
  const api = useMemo<ToastApi>(
    () => ({ success: (text) => push('success', text), error: (text) => push('danger', text) }),
    [push],
  )

  return (
    <ToastContext.Provider value={api}>
      {children}
      <div
        aria-live="polite"
        aria-label={t.common.notifications}
        role="region"
        className="pointer-events-none fixed inset-x-0 bottom-20 z-[60] flex flex-col items-center gap-2 px-4 md:bottom-6 md:items-end md:px-6"
      >
        {items.map((item) => {
          const Icon = item.tone === 'success' ? CheckCircle2 : AlertCircle
          return (
            <div
              key={item.id}
              className={cn(
                'pointer-events-auto flex w-full max-w-sm items-start gap-3 rounded-md border bg-surface px-4 py-3 text-[0.9375rem] text-ink shadow-float',
                'animate-in fade-in-0 slide-in-from-bottom-2',
                item.tone === 'success' ? 'border-forest/25' : 'border-danger/30',
              )}
            >
              <Icon
                className={cn('mt-0.5 size-5 shrink-0', item.tone === 'success' ? 'text-forest' : 'text-danger')}
                aria-hidden
              />
              <p className="min-w-0 flex-1">{item.text}</p>
              <button
                type="button"
                onClick={() => dismiss(item.id)}
                className="-m-1 grid size-7 shrink-0 place-items-center rounded-sm text-ink-muted hover:bg-sunken hover:text-ink"
              >
                <X className="size-4" aria-hidden />
                <span className="sr-only">{t.common.dismiss}</span>
              </button>
            </div>
          )
        })}
      </div>
    </ToastContext.Provider>
  )
}

const noop: ToastApi = { success: () => {}, error: () => {} }

export function useToast(): ToastApi {
  return useContext(ToastContext) ?? noop
}
