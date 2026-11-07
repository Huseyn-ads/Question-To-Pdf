import { useEffect, useState } from 'react'

/**
 * Remaining time derived from the server clock: the offset between
 * `server_time` and the local clock at the moment the response arrived is
 * applied to every local tick, so a wrong device clock does not matter.
 */
export function remainingMs(deadlineAt: string, serverTime: string, receivedAt: number, now: number): number {
  const offset = Date.parse(serverTime) - receivedAt
  return Date.parse(deadlineAt) - (now + offset)
}

export function useCountdown(
  deadlineAt: string | null | undefined,
  serverTime: string | undefined,
  receivedAt: number,
): number | null {
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    if (!deadlineAt || !serverTime) return
    let timer: ReturnType<typeof setTimeout>
    const schedule = (at: number) => {
      const value = remainingMs(deadlineAt, serverTime, receivedAt, at)
      // Align ticks to whole seconds of the remaining time.
      if (value > 0) timer = setTimeout(tick, (value % 1000) + 15)
    }
    const tick = () => {
      const at = Date.now()
      setNow(at)
      schedule(at)
    }
    timer = setTimeout(tick, 0)
    return () => clearTimeout(timer)
  }, [deadlineAt, serverTime, receivedAt])

  if (!deadlineAt || !serverTime) return null
  return Math.max(0, remainingMs(deadlineAt, serverTime, receivedAt, now))
}

export function formatClock(ms: number): string {
  const total = Math.ceil(ms / 1000)
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  const mm = String(m).padStart(h ? 2 : 1, '0')
  const ss = String(s).padStart(2, '0')
  return h ? `${h}:${mm}:${ss}` : `${mm}:${ss}`
}
