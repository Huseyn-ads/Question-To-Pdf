import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { setRequestLanguage } from '@/api/http'
import { ru, type Messages } from './ru'
import { az } from './az'
import { plural as pluralBase, type PluralForms } from './plural'
import { formatAzDate, formatAzRelativeDays, hasAzerbaijaniIntl } from './azDates'

export type Locale = 'az' | 'ru'
export const LOCALES: Locale[] = ['az', 'ru']
const dictionaries: Record<Locale, Messages> = { az, ru }

/** Shared with index.html (pre-paint script) and the backend email page. */
export const LOCALE_STORAGE_KEY = 'telebeland.lang'

function isLocale(value: unknown): value is Locale {
  return value === 'az' || value === 'ru'
}

/** Saved choice, then the browser languages; Russian when nothing matches. */
export function detectLocale(): Locale {
  try {
    const saved = localStorage.getItem(LOCALE_STORAGE_KEY)
    if (isLocale(saved)) return saved
  } catch {
    // Storage blocked: fall through to the browser languages.
  }
  const languages = typeof navigator === 'undefined' ? [] : (navigator.languages ?? [navigator.language])
  for (const tag of languages) {
    const primary = tag?.toLowerCase().split('-')[0]
    if (isLocale(primary)) return primary
  }
  return 'ru'
}

type I18n = {
  locale: Locale
  setLocale: (locale: Locale) => void
  t: Messages
  plural: (count: number, forms: PluralForms) => string
  formatDate: (iso: string, options?: Intl.DateTimeFormatOptions) => string
  formatRelativeDay: (iso: string, now?: Date) => string
  formatNumber: (value: number, options?: Intl.NumberFormatOptions) => string
}

const I18nContext = createContext<I18n | null>(null)

export function I18nProvider({ locale: initial, children }: { locale?: Locale; children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => initial ?? detectLocale())

  // Keep <html lang>, the API language header and the page title in sync.
  useEffect(() => {
    document.documentElement.lang = locale
    setRequestLanguage(locale)
  }, [locale])

  // Another tab switched the language.
  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key === LOCALE_STORAGE_KEY && isLocale(event.newValue)) setLocaleState(event.newValue)
    }
    window.addEventListener('storage', onStorage)
    return () => window.removeEventListener('storage', onStorage)
  }, [])

  const setLocale = useCallback((next: Locale) => {
    setLocaleState(next)
    setRequestLanguage(next)
    try {
      localStorage.setItem(LOCALE_STORAGE_KEY, next)
    } catch {
      // The choice still applies to this page.
    }
  }, [])

  const value = useMemo<I18n>(() => {
    const t = dictionaries[locale]
    const tag = t.meta.locale
    // Without Azerbaijani ICU data the browser would print English fallbacks.
    const manualAz = locale === 'az' && !hasAzerbaijaniIntl()
    const dateFmt = new Intl.DateTimeFormat(tag, { day: 'numeric', month: 'long' })
    const rtf = new Intl.RelativeTimeFormat(tag, { numeric: 'auto' })
    const formatDate = (iso: string, options?: Intl.DateTimeFormatOptions) => {
      const date = new Date(iso)
      if (manualAz) return formatAzDate(date, options)
      return options ? new Intl.DateTimeFormat(tag, options).format(date) : dateFmt.format(date)
    }
    return {
      locale,
      setLocale,
      t,
      plural: (count, forms) => pluralBase(tag, count, forms),
      formatDate,
      formatRelativeDay: (iso, now = new Date()) => {
        const date = new Date(iso)
        const startOf = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
        const days = Math.round((startOf(date) - startOf(now)) / 86_400_000)
        if (days > -7 && days <= 0) return manualAz ? formatAzRelativeDays(days) : rtf.format(days, 'day')
        return formatDate(iso)
      },
      formatNumber: (value, options) => new Intl.NumberFormat(tag, options).format(value),
    }
  }, [locale, setLocale])

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n(): I18n {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error('useI18n must be used inside I18nProvider')
  return ctx
}

export function useT(): Messages {
  return useI18n().t
}

export type { Messages, PluralForms }
