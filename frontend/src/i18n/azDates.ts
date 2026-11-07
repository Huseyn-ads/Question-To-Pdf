/**
 * Azerbaijani dates without Intl data. Some Chromium builds ship ICU without
 * Azerbaijani and silently fall back to "M11 7, Sat"; this covers the formats
 * the app uses (day + month, weekday, time, relative days).
 */
const MONTHS = ['yanvar', 'fevral', 'mart', 'aprel', 'may', 'iyun', 'iyul', 'avqust', 'sentyabr', 'oktyabr', 'noyabr', 'dekabr']
const WEEKDAYS = ['bazar', 'bazar ertəsi', 'çərşənbə axşamı', 'çərşənbə', 'cümə axşamı', 'cümə', 'şənbə']

let supported: boolean | undefined

/** True when the browser really formats Azerbaijani month names. */
export function hasAzerbaijaniIntl(): boolean {
  if (supported === undefined) {
    try {
      supported = new Intl.DateTimeFormat('az-AZ', { month: 'long' }).format(new Date(2020, 10, 15)) === 'noyabr'
    } catch {
      supported = false
    }
  }
  return supported
}

const pad = (n: number) => String(n).padStart(2, '0')

export function formatAzDate(date: Date, options: Intl.DateTimeFormatOptions = { day: 'numeric', month: 'long' }): string {
  const parts: string[] = []
  if (options.day || options.month) {
    parts.push([options.day ? String(date.getDate()) : '', options.month ? MONTHS[date.getMonth()] : ''].filter(Boolean).join(' '))
  }
  let text = parts.join('')
  if (options.weekday) text = text ? `${text}, ${WEEKDAYS[date.getDay()]}` : WEEKDAYS[date.getDay()]!
  if (options.hour) text = `${text} ${pad(date.getHours())}:${pad(date.getMinutes())}`.trim()
  return text
}

export function formatAzRelativeDays(days: number): string {
  if (days === 0) return 'bu gün'
  if (days === -1) return 'dünən'
  return days < 0 ? `${-days} gün əvvəl` : `${days} gün sonra`
}
