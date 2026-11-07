import { NavLink, Outlet } from 'react-router'
import { LogOut, Shapes } from 'lucide-react'
import { useLogout } from '@/features/auth/useLogout'
import { useMe } from '@/features/auth/session'
import { usePaths } from '@/app/paths'
import { useT } from '@/i18n'
import { cn } from '@/lib/utils'
import { Logo } from '@/components/brand/Logo'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { mainNav, tools } from './navigation'

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
  const t = useT()
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
    <aside className="sticky top-0 z-30 hidden h-dvh w-sidebar shrink-0 flex-col border-r border-line bg-[#EFE8DC] lg:flex">
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
        <div>
          <p className="px-3 pb-2 text-xs font-semibold tracking-wide text-ink-subtle uppercase">
            {t.nav.tools}
          </p>
          <ul className="flex flex-col gap-1">
            {tools(t, paths).map((tool) =>
              tool.to ? (
                <li key={tool.key}>
                  <NavLink to={tool.to} className={navLinkClass}>
                    <tool.icon className="size-[1.125rem]" aria-hidden />
                    {tool.shortTitle}
                  </NavLink>
                </li>
              ) : (
                <li
                  key={tool.key}
                  className="flex h-10 items-center gap-3 rounded-md px-3 text-[0.9375rem] text-ink-subtle"
                  aria-disabled="true"
                >
                  <tool.icon className="size-[1.125rem]" aria-hidden />
                  <span className="truncate">{tool.shortTitle}</span>
                  <Badge tone="outline" className="ml-auto">
                    {t.common.soon}
                  </Badge>
                </li>
              ),
            )}
          </ul>
        </div>
      </nav>
      <div className="border-t border-line p-3">
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
                  {item.label}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
        <div className="ml-auto">
          <UserMenu variant="header" />
        </div>
      </div>
    </header>
  )
}

function BottomNav() {
  const t = useT()
  const paths = usePaths()
  const items = [
    ...mainNav(t, paths),
    { key: 'tools', label: t.nav.tools, shortLabel: undefined, icon: Shapes, to: paths.importDocx, end: false },
  ]
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
        className="sr-only z-50 rounded-md bg-forest px-4 py-2 text-surface focus:not-sr-only focus:fixed focus:top-3 focus:left-3"
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
