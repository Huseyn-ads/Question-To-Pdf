import { NavLink, Outlet } from 'react-router'
import { Check, Home, Languages, LogOut, Monitor, Moon, Sun } from 'lucide-react'
import { useLogout } from '@/features/auth/useLogout'
import { useMe } from '@/features/auth/session'
import { usePaths } from '@/app/paths'
import { LOCALES, useI18n, useT } from '@/i18n'
import { useTheme } from '@/theme/theme'
import { cn } from '@/lib/utils'
import { Logo } from '@/components/brand/Logo'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { mainNav } from './navigation'
import { Preferences } from './Preferences'

function initials(name: string) {
  return (
    name
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((part) => part[0]?.toUpperCase())
      .join('') || '·'
  )
}

function UserMenu({ variant }: { variant: 'sidebar' | 'header' }) {
  const { t, locale, setLocale } = useI18n()
  const { preference, setPreference } = useTheme()
  const paths = usePaths()
  const { data: user } = useMe()
  const logout = useLogout()
  const name = user?.name ?? ''
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          'flex min-w-0 items-center gap-3 rounded-md text-left hover:bg-sunken/70',
          variant === 'sidebar' ? 'w-full p-2' : 'p-1',
        )}
        aria-label={t.nav.account}
      >
        <span className="grid size-9 shrink-0 place-items-center rounded-full bg-forest-soft text-sm font-semibold text-forest">
          {initials(name)}
        </span>
        {variant === 'sidebar' && (
          <span className="min-w-0">
            <span className="block truncate text-sm font-medium text-ink">{name || '—'}</span>
            <span className="block truncate text-xs text-ink-subtle">{user?.email}</span>
          </span>
        )}
      </DropdownMenuTrigger>
      <DropdownMenuContent align={variant === 'sidebar' ? 'start' : 'end'} className="w-60">
        <DropdownMenuLabel className="truncate font-normal text-ink-muted">{user?.email}</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem asChild>
          <NavLink to={paths.home}>
            <Home aria-hidden />
            {t.nav.home}
          </NavLink>
        </DropdownMenuItem>
        {/* On phones the header has no room for the switches: they live here. */}
        {variant === 'header' && (
          <div className="sm:hidden">
            <DropdownMenuSeparator />
            <DropdownMenuLabel className="flex items-center gap-2 font-normal text-ink-muted">
              <Languages className="size-4" aria-hidden />
              {t.settings.language}
            </DropdownMenuLabel>
            {LOCALES.map((code) => (
              <DropdownMenuItem
                key={code}
                lang={code}
                role="menuitemradio"
                aria-checked={locale === code}
                onSelect={(event) => {
                  event.preventDefault()
                  setLocale(code)
                }}
              >
                <span className="w-4 text-xs font-semibold uppercase">{code}</span>
                {code === 'az' ? 'Azərbaycanca' : 'Русский'}
                {locale === code && <Check className="ml-auto text-forest" aria-hidden />}
              </DropdownMenuItem>
            ))}
            <DropdownMenuSeparator />
            <DropdownMenuLabel className="font-normal text-ink-muted">{t.settings.theme}</DropdownMenuLabel>
            {(
              [
                ['light', Sun, t.settings.themeLight],
                ['dark', Moon, t.settings.themeDark],
                ['system', Monitor, t.settings.themeSystem],
              ] as const
            ).map(([value, Icon, label]) => (
              <DropdownMenuItem
                key={value}
                role="menuitemradio"
                aria-checked={preference === value}
                onSelect={(event) => {
                  event.preventDefault()
                  setPreference(value)
                }}
              >
                <Icon aria-hidden />
                {label}
                {preference === value && <Check className="ml-auto text-forest" aria-hidden />}
              </DropdownMenuItem>
            ))}
          </div>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem
          disabled={logout.isPending}
          onSelect={(event) => {
            event.preventDefault()
            if (!logout.isPending) logout.mutate()
          }}
        >
          <LogOut aria-hidden />
          {t.nav.logout}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    'group flex h-10 items-center gap-3 rounded-md px-3 text-[0.9375rem] transition-colors',
    isActive
      ? 'bg-surface font-semibold text-forest shadow-raise ring-1 ring-line'
      : 'text-ink-muted hover:bg-surface/60 hover:text-ink',
  )

function Sidebar() {
  const t = useT()
  const paths = usePaths()
  return (
    <aside className="sticky top-0 z-30 hidden h-dvh w-sidebar shrink-0 flex-col border-r border-line bg-rail lg:flex">
      <div className="flex h-18 items-center px-5">
        <NavLink to={paths.dashboard} className="rounded-md" aria-label={t.meta.appName}>
          <Logo />
        </NavLink>
      </div>
      <nav aria-label={t.nav.main} className="flex flex-1 flex-col gap-7 overflow-y-auto px-3 pt-2 pb-4">
        <ul className="flex flex-col gap-1">
          {mainNav(t, paths).map((item) => (
            <li key={item.key}>
              <NavLink to={item.to} end={item.end} className={navLinkClass}>
                <item.icon className="size-[1.125rem]" aria-hidden />
                {item.label}
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>
      <div className="flex flex-col gap-2 border-t border-line p-3">
        <Preferences className="px-2 pt-1" />
        <UserMenu variant="sidebar" />
      </div>
    </aside>
  )
}

function TopBar() {
  const t = useT()
  const paths = usePaths()
  return (
    <header className="sticky top-0 z-30 border-b border-line bg-paper/95 backdrop-blur-sm lg:hidden">
      <div className="mx-auto flex h-16 max-w-content items-center gap-4 px-4 md:px-8">
        <NavLink to={paths.dashboard} className="rounded-md" aria-label={t.meta.appName}>
          <Logo />
        </NavLink>
        <nav aria-label={t.nav.main} className="ml-4 hidden flex-1 md:block">
          <ul className="flex gap-1">
            {mainNav(t, paths).map((item) => (
              <li key={item.key}>
                <NavLink
                  to={item.to}
                  end={item.end}
                  className={({ isActive }) =>
                    cn(
                      'flex h-9 items-center rounded-md px-3 text-sm',
                      isActive ? 'bg-surface font-semibold text-forest ring-1 ring-line' : 'text-ink-muted hover:text-ink',
                    )
                  }
                >
                  {item.shortLabel ?? item.label}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
        <div className="ml-auto flex items-center gap-2">
          <Preferences className="hidden sm:flex" />
          <UserMenu variant="header" />
        </div>
      </div>
    </header>
  )
}

function BottomNav() {
  const t = useT()
  const paths = usePaths()
  const items = mainNav(t, paths)
  return (
    <nav
      aria-label={t.nav.main}
      className="pb-safe fixed inset-x-0 bottom-0 z-30 border-t border-line bg-surface/97 backdrop-blur-sm md:hidden"
    >
      <ul className="grid grid-cols-4">
        {items.map((item) => (
          <li key={item.key}>
            <NavLink
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                cn(
                  'flex flex-col items-center gap-1 pt-2.5 pb-1 text-[0.6875rem] leading-tight',
                  isActive ? 'font-semibold text-forest' : 'text-ink-muted',
                )
              }
            >
              {({ isActive }) => (
                <>
                  <span
                    className={cn(
                      'grid h-7 w-12 place-items-center rounded-full transition-colors',
                      isActive && 'bg-forest-soft',
                    )}
                  >
                    <item.icon className="size-5" aria-hidden />
                  </span>
                  <span className="max-w-full truncate px-1">{item.shortLabel ?? item.label}</span>
                </>
              )}
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  )
}

export function AppShell() {
  const t = useT()
  return (
    <div className="min-h-dvh lg:flex">
      <a
        href="#main"
        className="sr-only z-50 rounded-md bg-forest px-4 py-2 text-on-forest focus:not-sr-only focus:fixed focus:top-3 focus:left-3"
      >
        {t.nav.skipToContent}
      </a>
      <Sidebar />
      <TopBar />
      <div className="min-w-0 flex-1">
        <main id="main" tabIndex={-1} className="pb-24 outline-none md:pb-12">
          <Outlet />
        </main>
      </div>
      <BottomNav />
    </div>
  )
}
