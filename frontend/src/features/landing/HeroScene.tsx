import { useEffect, useRef } from 'react'
import { CheckCircle2, Clock3, FileText, Sparkles } from 'lucide-react'
import { useT } from '@/i18n'
import { optionLetter } from '@/features/attempt/optionLetter'

const RESULT = 92

/**
 * The hero's 3D still life: a DOCX file, a question card and a result card
 * floating at different depths. It tilts towards the pointer on devices with
 * a fine pointer; otherwise (and with reduced motion) it only floats or stays.
 * Purely decorative for assistive tech, with one text label.
 */
export function HeroScene() {
  const t = useT()
  const scene = t.landing.scene
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = ref.current
    if (!el) return
    const finePointer = window.matchMedia('(pointer: fine)').matches
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (!finePointer || reduced) return

    let frame = 0
    const onMove = (event: PointerEvent) => {
      cancelAnimationFrame(frame)
      frame = requestAnimationFrame(() => {
        const box = el.getBoundingClientRect()
        const x = (event.clientX - (box.left + box.width / 2)) / window.innerWidth
        const y = (event.clientY - (box.top + box.height / 2)) / window.innerHeight
        el.style.setProperty('--ry', `${Math.max(-1, Math.min(1, x)) * 16}deg`)
        el.style.setProperty('--rx', `${Math.max(-1, Math.min(1, y)) * -10}deg`)
        el.dataset.active = 'true'
      })
    }
    const onLeave = () => {
      el.style.setProperty('--ry', '0deg')
      el.style.setProperty('--rx', '0deg')
      el.dataset.active = 'false'
    }
    window.addEventListener('pointermove', onMove, { passive: true })
    document.documentElement.addEventListener('pointerleave', onLeave)
    return () => {
      cancelAnimationFrame(frame)
      window.removeEventListener('pointermove', onMove)
      document.documentElement.removeEventListener('pointerleave', onLeave)
    }
  }, [])

  const circumference = 2 * Math.PI * 26

  return (
    <div ref={ref} className="tb-scene" role="img" aria-label={scene.label}>
      <div className="tb-floor" aria-hidden />
      <div className="tb-rig" aria-hidden>
        {/* Back: the uploaded document */}
        <div className="tb-layer tb-file">
          <div className="flex items-center gap-[2.4cqi]">
            <span className="grid size-[max(2rem,8cqi)] shrink-0 place-items-center rounded-lg bg-sage-soft text-sage-ink">
              <FileText className="size-[55%]" />
            </span>
            <div className="min-w-0">
              <p className="truncate text-[max(0.7rem,2.6cqi)] font-semibold text-ink">{scene.file}</p>
              <p className="truncate text-[max(0.62rem,2.2cqi)] text-ink-subtle">{scene.fileMeta}</p>
            </div>
          </div>
          <div className="mt-[3.4cqi] space-y-[1.8cqi]">
            {[92, 76, 84, 58].map((w, i) => (
              <span key={i} className="block h-[max(0.35rem,1.3cqi)] rounded-full bg-sunken" style={{ width: `${w}%` }} />
            ))}
          </div>
        </div>

        {/* Middle: a question being answered */}
        <div className="tb-layer tb-question">
          <div className="flex items-center justify-between gap-2">
            <span className="text-[max(0.62rem,2.2cqi)] font-semibold tracking-wide text-forest uppercase">
              {scene.counter}
            </span>
            <span className="h-[max(0.3rem,1cqi)] w-[30%] overflow-hidden rounded-full bg-sunken">
              <span className="block h-full w-[15%] rounded-full bg-forest" />
            </span>
          </div>
          <p className="mt-[2.6cqi] font-display text-[max(0.9rem,3.6cqi)] leading-snug font-semibold text-ink">
            {scene.question}
          </p>
          <div className="mt-[3cqi] flex flex-col gap-[1.6cqi]">
            {scene.options.map((option, i) => (
              <div key={option} className="tb-option" data-right={i === 1}>
                <span className="tb-option-mark">{optionLetter(i)}</span>
                {option}
              </div>
            ))}
          </div>
          <p className="tb-verdict mt-[2.6cqi] flex items-center gap-[1.4cqi] text-[max(0.72rem,2.7cqi)] font-semibold text-forest">
            <CheckCircle2 className="size-[1.1em]" />
            {scene.verdict}
          </p>
        </div>

        {/* Front: the result */}
        <div className="tb-layer tb-result">
          <svg viewBox="0 0 64 64" className="mx-auto w-[64%]">
            <circle cx="32" cy="32" r="26" fill="none" stroke="var(--color-sunken)" strokeWidth="7" />
            <circle
              cx="32"
              cy="32"
              r="26"
              fill="none"
              stroke="var(--color-forest)"
              strokeWidth="7"
              strokeLinecap="round"
              strokeDasharray={`${(RESULT / 100) * circumference} ${circumference}`}
              transform="rotate(-90 32 32)"
            />
            <text
              x="32"
              y="36.5"
              textAnchor="middle"
              fontSize="14"
              fontWeight="700"
              fill="var(--color-ink)"
              fontFamily="var(--font-serif)"
            >
              {RESULT}%
            </text>
          </svg>
          <p className="mt-[1.6cqi] text-[max(0.62rem,2.3cqi)] font-medium text-ink-muted">{scene.result}</p>
        </div>

        <span className="tb-chip">
          <Sparkles className="size-[1.1em]" />
          {scene.streak}
        </span>
        <span className="tb-timer">
          <Clock3 className="size-[1.1em] text-forest" />
          {scene.timer}
        </span>
      </div>
    </div>
  )
}
