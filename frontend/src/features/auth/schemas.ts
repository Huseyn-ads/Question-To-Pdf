import { z } from 'zod'
import type { Messages } from '@/i18n'

const utf8Bytes = (value: string) => new TextEncoder().encode(value).length

/** Mirrors the backend rules: password ≥ 8 Unicode characters and ≤ 72 UTF-8 bytes. */
export function passwordSchema(t: Messages) {
  return z
    .string()
    .min(1, t.auth.validation.passwordRequired)
    .refine((value) => [...value].length >= 8, t.auth.validation.passwordShort)
    .refine((value) => utf8Bytes(value) <= 72, t.auth.validation.passwordLong)
}

export function emailSchema(t: Messages) {
  return z.string().trim().min(1, t.auth.validation.emailRequired).pipe(z.email(t.auth.validation.emailInvalid))
}

export function loginSchema(t: Messages) {
  return z.object({
    email: emailSchema(t),
    password: z.string().min(1, t.auth.validation.passwordRequired),
  })
}

export function registerSchema(t: Messages) {
  return z.object({
    name: z.string().trim().min(1, t.auth.validation.nameRequired).max(255, t.auth.validation.nameLong),
    email: emailSchema(t),
    password: passwordSchema(t),
  })
}

export type LoginValues = z.infer<ReturnType<typeof loginSchema>>
export type RegisterValues = z.infer<ReturnType<typeof registerSchema>>
