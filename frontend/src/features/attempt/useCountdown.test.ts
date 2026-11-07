import { describe, expect, it } from 'vitest'
import { formatClock, remainingMs } from './useCountdown'

describe('remainingMs', () => {
  it('uses the server clock, not the device clock', () => {
    // Device clock is 10 minutes fast relative to the server.
    const receivedAt = Date.parse('2026-11-02T10:40:05Z')
    const remaining = remainingMs('2026-11-02T11:00:00Z', '2026-11-02T10:30:05Z', receivedAt, receivedAt + 5_000)
    expect(remaining).toBe(29 * 60_000 + 50_000)
  })

  it('reaches zero exactly at the deadline (server_time >= deadline_at is expired)', () => {
    const receivedAt = 1_000_000
    expect(remainingMs('2026-11-02T11:00:00Z', '2026-11-02T10:59:59Z', receivedAt, receivedAt + 1_000)).toBe(0)
  })
})

describe('formatClock', () => {
  it('formats minutes and hours', () => {
    expect(formatClock(25 * 60_000)).toBe('25:00')
    expect(formatClock(59_001)).toBe('1:00')
    expect(formatClock(9_000)).toBe('0:09')
    expect(formatClock(3 * 3_600_000)).toBe('3:00:00')
  })
})
