import { describe, expect, it } from 'vitest'
import { formatAzDate, formatAzRelativeDays } from './azDates'

describe('Azerbaijani dates without Intl data', () => {
  const date = new Date(2026, 10, 7, 9, 5)

  it('formats the patterns the app uses', () => {
    expect(formatAzDate(date)).toBe('7 noyabr')
    expect(formatAzDate(date, { weekday: 'long', day: 'numeric', month: 'long' })).toBe('7 noyabr, şənbə')
    expect(formatAzDate(date, { day: 'numeric', month: 'long', hour: '2-digit', minute: '2-digit' })).toBe('7 noyabr 09:05')
  })

  it('names recent days', () => {
    expect(formatAzRelativeDays(0)).toBe('bu gün')
    expect(formatAzRelativeDays(-1)).toBe('dünən')
    expect(formatAzRelativeDays(-3)).toBe('3 gün əvvəl')
  })
})
