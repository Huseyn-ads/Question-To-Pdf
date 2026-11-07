import { useEffect } from 'react'
import { Link } from 'react-router'
import {
  ArrowRight,
  BarChart3,
  Check,
  CircleCheckBig,
  FileUp,
  LogIn,
  Timer,
  type LucideIcon,
} from 'lucide-react'
import { usePaths } from '@/app/paths'
import { useI18n } from '@/i18n'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Logo } from '@/components/brand/Logo'
import { LanguageSwitch, ThemeMenu } from '@/components/layout/Preferences'
import { tools } from '@/components/layout/navigation'
import { useMe } from '@/features/auth/session'
import { HeroScene } from './HeroScene'
import { Book3D } from './Book3D'
import './landing.css'

const FEATURE_ICONS: LucideIcon[] = [FileUp, CircleCheckBig, Timer, BarChart3]

export function LandingPage() {
  const { t } = useI18n()
  const me = useMe()
  const signedIn = Boolean(me.data)

  useEffect(() => {
    document.title = `telebeland — ${t.meta.tagline}`
    return () => {
      document.title = 'telebeland'
    }
  }, [t])

  return (
    <div className="min-h-dvh overflow-x-clip bg-paper">
      <a
        href="#main"
        className="sr-only z-50 rounded-md bg-forest px-4 py-2 text-on-forest focus:not-sr-only focus:fixed focus:top-3 focus:left-3"
      >
        {t.nav.skipToContent}
      </a>
      <LandingHeader signedIn={signedIn} pending={me.isPending} />
      <main id="main" tabIndex={-1} className="outline-none">
        <Hero signedIn={signedIn} />
        <Features />
        <HowItWorks />
        <Tools />
        <ClosingCta signedIn={signedIn} />
      </main>
      <LandingFooter />
    </div>
  )
}

function PrimaryCta({ signedIn, className, size = 'lg' }: { signedIn: boolean; className?: string; size?: 'md' | 'lg' }) {
  const { t } = useI18n()
  const paths = usePaths()
  return (
    <Button asChild size={size} className={className}>
      <Link to={signedIn ? paths.dashboard : paths.register}>
        {signedIn ? t.landing.openDashboard : t.landing.heroPrimary}
        <ArrowRight aria-hidden />
      </Link>
    </Button>
  )
}

function LandingHeader({ signedIn, pending }: { signedIn: boolean; pending: boolean }) {
  const { t } = useI18n()
  const paths = usePaths()
  const anchors = [
    { href: '#features', label: t.landing.navFeatures },
    { href: '#how', label: t.landing.navHow },
    { href: '#tools', label: t.landing.navTools },
  ]
  return (
    <header className="sticky top-0 z-40 border-b border-line/70 bg-paper/85 backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-[76rem] items-center gap-3 px-4 sm:px-6 lg:h-[4.5rem] lg:px-8">
        <Link to={paths.home} className="shrink-0 rounded-md" aria-label={t.meta.appName}>
          <Logo className="[&>span:last-child]:max-[379px]:hidden" />
        </Link>
        <nav aria-label={t.landing.navLabel} className="ml-6 hidden lg:block">
          <ul className="flex items-center gap-1">
            {anchors.map((anchor) => (
              <li key={anchor.href}>
                <a
                  href={anchor.href}
                  className="rounded-md px-3 py-2 text-[0.9375rem] text-ink-muted transition-colors hover:bg-sunken/60 hover:text-ink"
                >
                  {anchor.label}
                </a>
              </li>
            ))}
          </ul>
        </nav>
        <div className="ml-auto flex items-center gap-2">
          <LanguageSwitch />
          <ThemeMenu />
          <div className={cn('flex items-center gap-2', pending && 'invisible')}>
            {signedIn ? (
              <Button asChild size="sm" className="hidden sm:inline-flex">
                <Link to={paths.dashboard}>
                  {t.landing.openDashboard}
                  <ArrowRight aria-hidden />
                </Link>
              </Button>
            ) : (
              <>
                <Button asChild variant="ghost" size="sm" className="hidden sm:inline-flex">
                  <Link to={paths.login}>{t.landing.signIn}</Link>
                </Button>
                <Button asChild size="sm" className="hidden md:inline-flex">
                  <Link to={paths.register}>{t.landing.signUp}</Link>
                </Button>
              </>
            )}
            <Button asChild variant="secondary" size="icon-sm" className="rounded-full sm:hidden">
              <Link
                to={signedIn ? paths.dashboard : paths.login}
                aria-label={signedIn ? t.landing.openDashboard : t.landing.signIn}
              >
                {signedIn ? <ArrowRight aria-hidden /> : <LogIn aria-hidden />}
              </Link>
            </Button>
          </div>
        </div>
      </div>
    </header>
  )
}

function Hero({ signedIn }: { signedIn: boolean }) {
  const { t } = useI18n()
  return (
    <section aria-labelledby="hero-title" className="relative isolate">
      {/* Soft paper light: two blurred spots and a dot grid, no gradients. */}
      <div aria-hidden className="pointer-events-none absolute inset-0 -z-10 overflow-hidden">
        <span className="tb-blob -top-24 right-[-8rem] size-[30rem] bg-forest-soft opacity-80" />
        <span className="tb-blob bottom-[-10rem] left-[-10rem] size-[26rem] bg-terracotta-soft opacity-70" />
        <span className="tb-dots absolute inset-0 opacity-70" />
      </div>

      <div className="mx-auto grid max-w-[76rem] items-center gap-10 px-4 pt-10 pb-16 sm:px-6 md:pt-16 lg:grid-cols-[minmax(0,1.05fr)_minmax(0,1fr)] lg:gap-8 lg:px-8 lg:pt-20 lg:pb-24">
        <div className="max-w-2xl">
          <p className="inline-flex items-center gap-2 rounded-full border border-line bg-surface/80 px-3 py-1.5 text-sm font-medium text-ink-muted">
            <span className="size-2 rounded-full bg-terracotta" aria-hidden />
            {t.landing.heroEyebrow}
          </p>
          <h1
            id="hero-title"
            className="mt-5 font-display text-[2.375rem] leading-[1.08] font-semibold tracking-[-0.02em] text-ink sm:text-[3.25rem] xl:text-[3.875rem]"
          >
            {t.landing.heroTitle}{' '}
            <span className="relative inline-block text-forest italic">
              {t.landing.heroTitleAccent}
              <svg
                aria-hidden
                viewBox="0 0 300 14"
                preserveAspectRatio="none"
                className="absolute -bottom-2 left-0 h-3 w-full text-terracotta"
              >
                <path
                  d="M3 10c48-6 98-8 148-7s98 4 146 1"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="3.5"
                  strokeLinecap="round"
                />
              </svg>
            </span>
          </h1>
          <p className="mt-7 max-w-xl text-[1.0625rem] leading-[1.75] text-ink-muted sm:text-lg">{t.landing.heroText}</p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center">
            <PrimaryCta signedIn={signedIn} className="w-full sm:w-auto" />
            <Button asChild variant="secondary" size="lg" className="w-full sm:w-auto">
              <a href="#how">{t.landing.heroSecondary}</a>
            </Button>
          </div>
          {!signedIn && <p className="mt-4 text-sm text-ink-subtle">{t.landing.heroNote}</p>}
          <ul className="mt-8 flex flex-wrap gap-x-6 gap-y-3">
            {t.landing.highlights.map((item) => (
              <li key={item} className="flex items-center gap-2 text-[0.9375rem] font-medium text-ink">
                <span className="grid size-5 place-items-center rounded-full bg-forest-soft text-forest">
                  <Check className="size-3.5" strokeWidth={3} aria-hidden />
                </span>
                {item}
              </li>
            ))}
          </ul>
        </div>

        <div className="relative mx-auto w-full max-w-[30rem] py-4 sm:max-w-[34rem] lg:max-w-none">
          <HeroScene />
        </div>
      </div>
    </section>
  )
}

function SectionIntro({ id, eyebrow, title, text }: { id: string; eyebrow: string; title: string; text?: string }) {
  return (
    <div className="max-w-2xl">
      <p className="text-sm font-semibold tracking-[0.08em] text-terracotta-ink uppercase">{eyebrow}</p>
      <h2 id={id} className="mt-3 font-display text-[1.875rem] leading-tight font-semibold text-ink sm:text-[2.375rem]">
        {title}
      </h2>
      {text && <p className="mt-4 text-[1.0625rem] leading-relaxed text-ink-muted">{text}</p>}
    </div>
  )
}

function Features() {
  const { t } = useI18n()
  const [first, ...rest] = t.landing.features
  const FirstIcon = FEATURE_ICONS[0]!
  return (
    <section id="features" aria-labelledby="features-title" className="scroll-mt-20 border-t border-line/70 bg-surface/60">
      <div className="mx-auto max-w-[76rem] px-4 py-16 sm:px-6 md:py-24 lg:px-8">
        <SectionIntro
          id="features-title"
          eyebrow={t.landing.navFeatures}
          title={t.landing.featuresTitle}
          text={t.landing.featuresText}
        />
        <div className="mt-12 grid gap-4 md:grid-cols-2 lg:grid-cols-[1.15fr_1fr] lg:gap-5">
          {/* Lead feature with a small document → questions visual */}
          {first && (
            <article className="relative overflow-hidden rounded-xl border border-forest/15 bg-forest-tint p-6 sm:p-8 md:row-span-3 md:flex md:flex-col">
              <span className="grid size-12 place-items-center rounded-lg bg-forest text-on-forest">
                <FirstIcon className="size-6" aria-hidden />
              </span>
              <h3 className="mt-5 font-display text-2xl font-semibold text-ink">{first.title}</h3>
              <p className="mt-3 max-w-md text-[1.0625rem] leading-relaxed text-ink-muted">{first.text}</p>
              <DocToQuestions />
            </article>
          )}
          {rest.map((feature, i) => {
            const Icon = FEATURE_ICONS[i + 1]!
            return (
              <article key={feature.title} className="flex gap-4 rounded-xl border border-line bg-surface p-5 sm:p-6">
                <span
                  className={cn(
                    'grid size-11 shrink-0 place-items-center rounded-lg',
                    i === 0 && 'bg-forest-soft text-forest',
                    i === 1 && 'bg-terracotta-soft text-terracotta-ink',
                    i === 2 && 'bg-sage-soft text-sage-ink',
                  )}
                >
                  <Icon className="size-5" aria-hidden />
                </span>
                <div className="min-w-0">
                  <h3 className="text-lg font-semibold text-ink">{feature.title}</h3>
                  <p className="mt-1.5 leading-relaxed text-ink-muted">{feature.text}</p>
                </div>
              </article>
            )
          })}
        </div>
      </div>
    </section>
  )
}

/** Lines of a document turning into answer options. Decorative. */
function DocToQuestions() {
  return (
    <div aria-hidden className="mt-8 flex items-center gap-4 md:mt-auto md:pt-8">
      <div className="w-[42%] rounded-lg border border-line bg-surface p-3.5 shadow-raise">
        <span className="mb-2.5 block h-2 w-1/2 rounded-full bg-sage/60" />
        {[100, 86, 92, 70, 80].map((w, i) => (
          <span key={i} className="mt-1.5 block h-1.5 rounded-full bg-sunken" style={{ width: `${w}%` }} />
        ))}
      </div>
      <ArrowRight className="size-5 shrink-0 text-forest" />
      <div className="flex-1 space-y-2">
        {['A', 'B', 'C'].map((letter, i) => (
          <div
            key={letter}
            className={cn(
              'flex items-center gap-2.5 rounded-md border bg-surface px-2.5 py-2',
              i === 1 ? 'border-forest' : 'border-line',
            )}
          >
            <span
              className={cn(
                'grid size-5 shrink-0 place-items-center rounded-full text-[0.625rem] font-semibold',
                i === 1 ? 'bg-forest text-on-forest' : 'border border-control/60 text-ink-muted',
              )}
            >
              {i === 1 ? <Check className="size-3" strokeWidth={3} /> : letter}
            </span>
            <span className="h-1.5 flex-1 rounded-full bg-sunken" />
          </div>
        ))}
      </div>
    </div>
  )
}

function HowItWorks() {
  const { t } = useI18n()
  return (
    <section id="how" aria-labelledby="how-title" className="scroll-mt-20">
      <div className="mx-auto max-w-[76rem] px-4 py-16 sm:px-6 md:py-24 lg:px-8">
        <SectionIntro id="how-title" eyebrow={t.landing.navHow} title={t.landing.howTitle} />
        <ol className="relative mt-12 grid gap-8 md:grid-cols-3 md:gap-6">
          {/* Dashed path between the steps */}
          <span
            aria-hidden
            className="absolute top-7 right-[16%] left-[16%] hidden border-t-2 border-dashed border-line md:block"
          />
          {t.landing.steps.map((step, i) => (
            <li key={step.title} className="relative flex gap-4 md:flex-col md:items-center md:text-center">
              <span
                className={cn(
                  'relative grid size-14 shrink-0 place-items-center rounded-full border-4 border-paper font-display text-2xl font-semibold',
                  i === 2 ? 'bg-forest text-on-forest' : 'bg-surface text-forest ring-1 ring-line',
                )}
              >
                {i + 1}
              </span>
              <div className="md:mt-2">
                <h3 className="text-lg font-semibold text-ink">{step.title}</h3>
                <p className="mt-1.5 leading-relaxed text-ink-muted md:mx-auto md:max-w-xs">{step.text}</p>
              </div>
            </li>
          ))}
        </ol>
      </div>
    </section>
  )
}

function Tools() {
  const { t } = useI18n()
  const paths = usePaths()
  return (
    <section id="tools" aria-labelledby="tools-title" className="scroll-mt-20 border-y border-line/70 bg-rail/70">
      <div className="mx-auto max-w-[76rem] px-4 py-16 sm:px-6 md:py-20 lg:px-8">
        <SectionIntro id="tools-title" eyebrow={t.landing.navTools} title={t.landing.toolsTitle} text={t.landing.toolsText} />
        <ul className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {tools(t, paths).map((tool) => {
            const available = tool.to !== null
            const body = (
              <>
                <div className="flex items-start justify-between gap-3">
                  <span
                    className={cn(
                      'grid size-11 place-items-center rounded-lg',
                      available ? 'bg-forest text-on-forest' : 'bg-sunken text-ink-muted',
                    )}
                  >
                    <tool.icon className="size-5" aria-hidden />
                  </span>
                  <Badge tone={available ? 'forest' : 'outline'}>
                    {available ? t.landing.toolAvailable : t.landing.toolSoon}
                  </Badge>
                </div>
                <h3 className="mt-5 text-lg font-semibold text-ink">{tool.title}</h3>
                <p className="mt-1.5 leading-relaxed text-ink-muted">{tool.description}</p>
                {available && (
                  <span className="mt-4 inline-flex items-center gap-1.5 text-sm font-semibold text-forest">
                    {t.landing.toolOpen}
                    <ArrowRight className="size-4 transition-transform group-hover:translate-x-0.5" aria-hidden />
                  </span>
                )}
              </>
            )
            return (
              <li key={tool.key} className={cn(!available && 'sm:last:col-span-2 lg:last:col-span-1')}>
                {tool.to ? (
                  <Link
                    to={tool.to}
                    className="group flex h-full flex-col rounded-xl border border-line bg-surface p-5 transition-[border-color,box-shadow] hover:border-forest/40 hover:shadow-float sm:p-6"
                  >
                    {body}
                  </Link>
                ) : (
                  <div className="flex h-full flex-col rounded-xl border border-dashed border-control/50 bg-surface/50 p-5 sm:p-6">
                    {body}
                  </div>
                )}
              </li>
            )
          })}
        </ul>
      </div>
    </section>
  )
}

function ClosingCta({ signedIn }: { signedIn: boolean }) {
  const { t } = useI18n()
  const paths = usePaths()
  return (
    <section aria-labelledby="cta-title" className="px-4 py-16 sm:px-6 md:py-24 lg:px-8">
      <div className="relative mx-auto grid max-w-[76rem] items-center gap-6 overflow-hidden rounded-[1.75rem] bg-forest px-6 py-12 text-on-forest sm:px-10 md:grid-cols-[minmax(0,1fr)_16rem] md:py-14 lg:px-16">
        <span aria-hidden className="tb-blob -top-24 -right-16 size-72 bg-on-forest opacity-10" />
        <div className="relative">
          <h2 id="cta-title" className="font-display text-[1.875rem] leading-tight font-semibold sm:text-[2.5rem]">
            {t.landing.ctaTitle}
          </h2>
          <p className="mt-4 max-w-lg text-[1.0625rem] leading-relaxed opacity-85">{t.landing.ctaText}</p>
          <Button
            asChild
            size="lg"
            className="mt-8 w-full bg-on-forest text-forest hover:bg-on-forest/90 active:bg-on-forest/90 sm:w-auto"
          >
            <Link to={signedIn ? paths.dashboard : paths.register}>
              {signedIn ? t.landing.openDashboard : t.landing.heroPrimary}
              <ArrowRight aria-hidden />
            </Link>
          </Button>
        </div>
        <div className="relative hidden h-64 md:block">
          <Book3D />
        </div>
      </div>
    </section>
  )
}

function LandingFooter() {
  const { t } = useI18n()
  const paths = usePaths()
  return (
    <footer className="border-t border-line/70">
      <div className="mx-auto flex max-w-[76rem] flex-col gap-6 px-4 py-10 sm:px-6 md:flex-row md:items-center md:justify-between lg:px-8">
        <div>
          <Logo />
          <p className="mt-2 text-sm text-ink-subtle">{t.landing.footerTagline}</p>
        </div>
        <nav aria-label={t.landing.navLabel} className="flex flex-wrap gap-x-6 gap-y-2 text-sm">
          <a href="#features" className="text-ink-muted hover:text-ink">
            {t.landing.navFeatures}
          </a>
          <a href="#tools" className="text-ink-muted hover:text-ink">
            {t.landing.navTools}
          </a>
          <Link to={paths.login} className="text-ink-muted hover:text-ink">
            {t.landing.signIn}
          </Link>
          <Link to={paths.register} className="text-ink-muted hover:text-ink">
            {t.auth.toRegister}
          </Link>
        </nav>
        <p className="text-sm text-ink-subtle">{t.landing.footerRights(new Date().getFullYear())}</p>
      </div>
    </footer>
  )
}
