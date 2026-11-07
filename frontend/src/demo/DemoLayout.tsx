import { useState } from 'react'
import { Outlet } from 'react-router'
import { QueryClientProvider } from '@tanstack/react-query'
import { FlaskConical, SlidersHorizontal } from 'lucide-react'
import { ApiProvider } from '@/api/context'
import { PathsProvider } from '@/app/paths'
import { createQueryClient } from '@/app/queryClient'
import { useT } from '@/i18n'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { demoApi, getDemoSettings, resetDemo, type DemoScenario, type DemoSettings } from './demoApi'

/**
 * Design-preview wrapper: in-memory API, its own query cache, /demo links
 * and a persistent banner so demo data is never mistaken for real data.
 */
export function DemoLayout() {
  const [queryClient] = useState(createQueryClient)
  return (
    <ApiProvider api={demoApi}>
      <PathsProvider base="/demo">
        <QueryClientProvider client={queryClient}>
          <DemoBanner />
          <Outlet />
        </QueryClientProvider>
      </PathsProvider>
    </ApiProvider>
  )
}

function DemoBanner() {
  const t = useT()
  const [open, setOpen] = useState(false)
  return (
    <>
      <div className="relative z-40 border-b border-warning-ink/20 bg-warning-soft text-warning-ink">
        <div className="mx-auto flex min-h-9 max-w-[90rem] items-center gap-2 px-3 py-1 text-xs sm:text-sm">
          <FlaskConical className="size-4 shrink-0" aria-hidden />
          <span className="min-w-0 flex-1 font-medium">
            <span className="sm:hidden">{t.demo.bannerShort}</span>
            <span className="hidden sm:inline">{t.demo.banner}</span>
          </span>
          <button
            type="button"
            onClick={() => setOpen(true)}
            className="inline-flex shrink-0 items-center gap-1.5 rounded-sm px-2 py-1 font-semibold underline decoration-warning-ink/40 underline-offset-4 hover:decoration-warning-ink"
          >
            <SlidersHorizontal className="size-3.5" aria-hidden />
            {t.demo.settings}
          </button>
        </div>
      </div>
      <DemoSettingsDialog open={open} onOpenChange={setOpen} />
    </>
  )
}

function DemoSettingsDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const t = useT()
  const [draft, setDraft] = useState<DemoSettings>(getDemoSettings)
  const scenarios: { value: DemoScenario; label: string }[] = [
    { value: 'timed', label: t.demo.scenarioTimed },
    { value: 'short', label: t.demo.scenarioShort },
    { value: 'unlimited', label: t.demo.scenarioUnlimited },
  ]
  const toggles: { key: 'failSaves' | 'slowSaves' | 'empty'; label: string }[] = [
    { key: 'slowSaves', label: t.demo.slowSaves },
    { key: 'failSaves', label: t.demo.failSaves },
    { key: 'empty', label: t.demo.emptyDashboard },
  ]
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (next) setDraft(getDemoSettings())
        onOpenChange(next)
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t.demo.settingsTitle}</DialogTitle>
          <DialogDescription>{t.demo.settingsText}</DialogDescription>
        </DialogHeader>
        <fieldset>
          <legend className="mb-2 text-sm font-semibold text-ink">{t.demo.scenario}</legend>
          <div className="flex flex-col gap-1.5">
            {scenarios.map((s) => (
              <label key={s.value} className="flex cursor-pointer items-center gap-3 rounded-md px-1 py-1.5 text-[0.9375rem]">
                <input
                  type="radio"
                  name="demo-scenario"
                  className="size-4 accent-forest"
                  checked={draft.scenario === s.value}
                  onChange={() => setDraft({ ...draft, scenario: s.value })}
                />
                {s.label}
              </label>
            ))}
          </div>
        </fieldset>
        <div className="flex flex-col gap-1.5 border-t border-line-soft pt-4">
          {toggles.map((toggle) => (
            <label key={toggle.key} className="flex cursor-pointer items-center gap-3 rounded-md px-1 py-1.5 text-[0.9375rem]">
              <input
                type="checkbox"
                className="size-4 accent-forest"
                checked={draft[toggle.key]}
                onChange={(e) => setDraft({ ...draft, [toggle.key]: e.target.checked })}
              />
              {toggle.label}
            </label>
          ))}
        </div>
        <DialogFooter>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            {t.common.cancel}
          </Button>
          <Button
            onClick={() => {
              resetDemo(draft)
              window.location.assign('/demo')
            }}
          >
            {t.demo.apply}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
