import { Check, Monitor, Moon, Sun } from 'lucide-react'
import { LOCALES, useI18n } from '@/i18n'
import { useTheme, type ThemePreference } from '@/theme/theme'
import { cn } from '@/lib/utils'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

/** AZ | RU segmented switch. Each button names its language for screen readers. */
export function LanguageSwitch({ className }: { className?: string }) {
  const { locale, setLocale, t } = useI18n()
  return (
    <div
      role="group"
      aria-label={t.settings.language}
      className={cn('inline-flex h-9 shrink-0 items-center rounded-full border border-line bg-surface p-0.5', className)}
    >
      {LOCALES.map((code) => (
        <button
          key={code}
          type="button"
          lang={code}
          aria-pressed={locale === code}
          aria-label={code === 'az' ? 'Azərbaycanca' : 'Русский'}
          onClick={() => setLocale(code)}
          className={cn(
            'h-full min-w-10 rounded-full px-2.5 text-xs font-semibold tracking-wide uppercase transition-colors',
            locale === code ? 'bg-forest text-on-forest' : 'text-ink-muted hover:text-ink',
          )}
        >
          {code}
        </button>
      ))}
    </div>
  )
}

const THEME_ICONS = { light: Sun, dark: Moon, system: Monitor } as const

/** Light / dark / system. The trigger shows the theme in effect right now. */
export function ThemeMenu({ className, align = 'end' }: { className?: string; align?: 'start' | 'end' }) {
  const { t } = useI18n()
  const { preference, theme, setPreference } = useTheme()
  const Icon = theme === 'dark' ? Moon : Sun
  const options: { value: ThemePreference; label: string }[] = [
    { value: 'light', label: t.settings.themeLight },
    { value: 'dark', label: t.settings.themeDark },
    { value: 'system', label: t.settings.themeSystem },
  ]
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={`${t.settings.theme}: ${options.find((o) => o.value === preference)?.label}`}
        className={cn(
          'grid size-9 shrink-0 place-items-center rounded-full border border-line bg-surface text-ink-muted transition-colors hover:text-ink',
          className,
        )}
      >
        <Icon className="size-[1.05rem]" aria-hidden />
      </DropdownMenuTrigger>
      <DropdownMenuContent align={align} className="w-48">
        <DropdownMenuLabel className="font-normal text-ink-muted">{t.settings.theme}</DropdownMenuLabel>
        {options.map((option) => {
          const OptionIcon = THEME_ICONS[option.value]
          return (
            <DropdownMenuItem
              key={option.value}
              onSelect={() => setPreference(option.value)}
              aria-checked={preference === option.value}
              role="menuitemradio"
            >
              <OptionIcon aria-hidden />
              {option.label}
              {preference === option.value && <Check className="ml-auto text-forest" aria-hidden />}
            </DropdownMenuItem>
          )
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export function Preferences({ className }: { className?: string }) {
  return (
    <div className={cn('flex items-center gap-2', className)}>
      <LanguageSwitch />
      <ThemeMenu />
    </div>
  )
}
