export type PluralForms = {
  one: string
  few?: string
  many?: string
  other: string
}

const rulesCache = new Map<string, Intl.PluralRules>()

function rules(locale: string) {
  let r = rulesCache.get(locale)
  if (!r) {
    r = new Intl.PluralRules(locale)
    rulesCache.set(locale, r)
  }
  return r
}

/**
 * Picks a plural form using CLDR rules for the locale.
 * Russian uses one/few/many ("1 вопрос", "3 вопроса", "5 вопросов");
 * Azerbaijani uses one/other. `#` in the form is replaced with the number.
 */
export function plural(locale: string, count: number, forms: PluralForms): string {
  const category = rules(locale).select(count) as keyof PluralForms
  const template = forms[category] ?? forms.other
  return template.replace('#', new Intl.NumberFormat(locale).format(count))
}
