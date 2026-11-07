import { createContext, useContext, useMemo, type ReactNode } from 'react'
import { ru, type Messages } from './ru'
import { plural as pluralBase, type PluralForms } from './plural'

export type Locale = 'ru'
// Add 'az' here together with `az.ts` typed as `Messages`.
const dictionaries: Record<Locale, Messages> = { ru }

type I18n = {
  locale: Locale
  t: Messages
  plural: (count: number, forms: PluralForms) => string
  formatDate: (iso: string, options?: Intl.DateTimeFormatOptions) => string
  formatRelativeDay: (iso: string, now?: Date) => string
  formatNumber: (value: number, options?: Intl.NumberFormatOptions) => string
}

const I18nContext = createContext<I18n | null>(null)

export function I18nProvider({ locale = 'ru', children }: { locale?: Locale; children: ReactNode }) {
  const value = useMemo<I18n>(() => {
    const t = dictionaries[locale]
    const tag = t.meta.locale
    const dateFmt = new Intl.DateTimeFormat(tag, { day: 'numeric', month: 'long' })
    const rtf = new Intl.RelativeTimeFormat(tag, { numeric: 'auto' })
    return {
      locale,
      t,
      plural: (count, forms) => pluralBase(tag, count, forms),
      formatDate: (iso, options) =>
        options ? new Intl.DateTimeFormat(tag, options).format(new Date(iso)) : dateFmt.format(new Date(iso)),
      formatRelativeDay: (iso, now = new Date()) => {
        const date = new Date(iso)
        const startOf = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
        const days = Math.round((startOf(date) - startOf(now)) / 86_400_000)
        if (days >= -1 && days <= 0) return rtf.format(days, 'day')
        if (days > -7 && days < 0) return rtf.format(days, 'day')
        return dateFmt.format(date)
      },
      formatNumber: (value, options) => new Intl.NumberFormat(tag, options).format(value),
    }
  }, [locale])

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
