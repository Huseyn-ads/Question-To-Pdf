import type { ReactNode } from 'react'
import { Link } from 'react-router'
import { useT } from '@/i18n'
import { Logo } from '@/components/brand/Logo'
import { DeskIllustration } from '@/components/brand/Illustrations'
import { Preferences } from '@/components/layout/Preferences'

/** Calm two-column frame for sign-in screens; a single column on phones. */
export function AuthLayout({ title, subtitle, children }: { title: string; subtitle?: ReactNode; children: ReactNode }) {
  const t = useT()
  return (
    <div className="min-h-dvh lg:grid lg:grid-cols-[minmax(0,5fr)_minmax(0,6fr)]">
      <aside className="relative hidden flex-col justify-between overflow-hidden border-r border-line bg-rail p-10 lg:flex xl:p-14">
        <Link to="/" className="w-fit rounded-md" aria-label={t.meta.appName}>
          <Logo />
        </Link>
        <div className="max-w-md">
          <p className="font-display text-[2.25rem] leading-tight font-semibold text-ink">{t.auth.brandTitle}</p>
          <p className="mt-4 text-reading text-ink-muted">{t.auth.brandText}</p>
        </div>
        <DeskIllustration className="w-full max-w-sm self-start" />
      </aside>

      <main id="main" className="relative flex min-h-dvh flex-col px-4 py-5 sm:px-8 lg:min-h-0 lg:justify-center lg:py-12">
        <div className="mb-10 flex items-center justify-between gap-3 lg:absolute lg:top-6 lg:right-8 lg:mb-0">
          <Link to="/" className="w-fit rounded-md lg:hidden" aria-label={t.meta.appName}>
            <Logo />
          </Link>
          <Preferences />
        </div>
        <div className="mx-auto w-full max-w-[26rem]">
          <h1 className="font-display text-[1.875rem] leading-tight font-semibold text-ink">{title}</h1>
          {subtitle && <div className="mt-2 text-reading text-ink-muted">{subtitle}</div>}
          <div className="mt-8">{children}</div>
        </div>
      </main>
    </div>
  )
}

export function AuthDivider() {
  const t = useT()
  return (
    <div className="my-6 flex items-center gap-3 text-sm text-ink-subtle" role="separator">
      <span className="h-px flex-1 bg-line" />
      {t.common.or}
      <span className="h-px flex-1 bg-line" />
    </div>
  )
}

function GoogleMark() {
  return (
    <svg viewBox="0 0 24 24" className="size-5" aria-hidden>
      <path
        fill="#4285F4"
        d="M23.5 12.27c0-.85-.08-1.66-.22-2.45H12v4.63h6.46a5.52 5.52 0 0 1-2.4 3.62v3h3.88c2.27-2.09 3.56-5.17 3.56-8.8Z"
      />
      <path
        fill="#34A853"
        d="M12 24c3.24 0 5.96-1.07 7.94-2.91l-3.88-3c-1.07.72-2.45 1.15-4.06 1.15-3.12 0-5.77-2.11-6.71-4.95H1.28v3.11A12 12 0 0 0 12 24Z"
      />
      <path fill="#FBBC05" d="M5.29 14.29a7.2 7.2 0 0 1 0-4.58V6.6H1.28a12 12 0 0 0 0 10.8l4.01-3.11Z" />
      <path
        fill="#EA4335"
        d="M12 4.75c1.76 0 3.34.61 4.59 1.8l3.44-3.44A11.94 11.94 0 0 0 12 0 12 12 0 0 0 1.28 6.6l4.01 3.11C6.23 6.86 8.88 4.75 12 4.75Z"
      />
    </svg>
  )
}

/**
 * Google sign-in is a full browser navigation to the backend route, never a
 * fetch: the backend sets its state cookie and redirects to Google.
 */
export function GoogleButton({ href, label, onBeforeNavigate }: { href: string; label: string; onBeforeNavigate: () => void }) {
  return (
    <a
      href={href}
      onClick={onBeforeNavigate}
      className="inline-flex h-12 w-full items-center justify-center gap-3 rounded-md border border-control/70 bg-surface px-5 text-base font-medium text-ink transition-colors hover:border-control hover:bg-forest-tint"
    >
      <GoogleMark />
      {label}
    </a>
  )
}
