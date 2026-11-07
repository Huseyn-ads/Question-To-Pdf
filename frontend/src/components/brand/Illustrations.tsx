import { cn } from '@/lib/utils'

const ink = 'var(--color-ink)'

/** Desk still life for the greeting: books with a sprig, an open notebook, a pencil and a mug. */
export function DeskIllustration({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 260 160" aria-hidden className={cn('h-auto w-full', className)} fill="none">
      <path d="M8 140.5h244" stroke={ink} strokeOpacity=".35" strokeWidth="1.5" strokeLinecap="round" />
      {/* books */}
      <rect x="22" y="116" width="86" height="24" rx="3" fill="var(--color-sage-soft)" stroke={ink} strokeWidth="1.5" />
      <path d="M31 116v24" stroke={ink} strokeWidth="1.5" />
      <path d="M40 128h56" stroke="var(--color-sage)" strokeWidth="1.5" strokeLinecap="round" />
      <rect x="30" y="96" width="72" height="20" rx="3" fill="var(--color-terracotta-soft)" stroke={ink} strokeWidth="1.5" />
      <path d="M94 96v20" stroke={ink} strokeWidth="1.5" />
      {/* sprig in a pot */}
      <path d="M55 96l-3-14h22l-3 14z" fill="var(--color-surface)" stroke={ink} strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M63 82c0-9 1-16 4-22" stroke={ink} strokeWidth="1.5" strokeLinecap="round" />
      <path d="M64 70c-8-1-13-6-14-13 8 0 13 5 14 13z" fill="var(--color-sage)" stroke={ink} strokeWidth="1.2" />
      <path d="M66 64c4-8 10-11 17-11-1 8-8 12-17 11z" fill="#A9BFA9" stroke={ink} strokeWidth="1.2" />
      {/* open notebook */}
      <path d="M118 140l12-22 36 3-5 19z" fill="var(--color-surface)" stroke={ink} strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M161 140l5-19 34-3 9 22z" fill="var(--color-surface)" stroke={ink} strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M131 126l26 2M128 132l27 2M170 127l24-2M171 133l26-2" stroke="var(--color-sage)" strokeWidth="1.2" strokeLinecap="round" />
      {/* pencil */}
      <g transform="rotate(-14 170 110)">
        <rect x="138" y="106" width="52" height="7" rx="1.5" fill="#D9895F" stroke={ink} strokeWidth="1.3" />
        <rect x="133" y="106" width="6" height="7" rx="1" fill="var(--color-sage-soft)" stroke={ink} strokeWidth="1.3" />
        <path d="M190 106l10 3.5-10 3.5z" fill="var(--color-terracotta-soft)" stroke={ink} strokeWidth="1.3" strokeLinejoin="round" />
      </g>
      {/* mug */}
      <path d="M214 110h26v22a8 8 0 01-8 8h-10a8 8 0 01-8-8z" fill="var(--color-forest-soft)" stroke={ink} strokeWidth="1.5" />
      <path d="M240 116a6.5 6.5 0 010 13" stroke={ink} strokeWidth="1.5" strokeLinecap="round" />
      <path d="M223 102c-3-4 3-6 0-10M231 102c-3-4 3-6 0-10" stroke={ink} strokeOpacity=".45" strokeWidth="1.4" strokeLinecap="round" />
    </svg>
  )
}

/** A blank sheet with a folded corner and pencil — for empty lists. */
export function EmptySheetIllustration({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 120 96" aria-hidden className={cn('h-auto w-28', className)} fill="none">
      <path d="M10 88.5h100" stroke={ink} strokeOpacity=".3" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M34 12h36l14 14v62H34z" fill="var(--color-surface)" stroke={ink} strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M70 12v14h14" fill="var(--color-sunken)" stroke={ink} strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M42 40h32M42 50h28M42 60h20" stroke="var(--color-sage)" strokeWidth="1.5" strokeLinecap="round" />
      <g transform="rotate(32 86 62)">
        <rect x="68" y="58" width="36" height="6" rx="1.5" fill="#D9895F" stroke={ink} strokeWidth="1.2" />
        <path d="M104 58l8 3-8 3z" fill="var(--color-terracotta-soft)" stroke={ink} strokeWidth="1.2" strokeLinejoin="round" />
      </g>
    </svg>
  )
}
