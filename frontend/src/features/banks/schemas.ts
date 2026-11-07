import { z } from 'zod'
import type { Messages } from '@/i18n'

export const BANK_TITLE_MAX = 150

export function bankSchema(t: Messages) {
  return z.object({
    title: z
      .string()
      .trim()
      .min(1, t.banks.titleRequired)
      .refine((value) => [...value].length <= BANK_TITLE_MAX, t.banks.titleLong(BANK_TITLE_MAX)),
    description: z.string().trim(),
  })
}
