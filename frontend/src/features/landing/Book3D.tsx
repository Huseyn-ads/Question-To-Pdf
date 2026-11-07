import { LogoMark } from '@/components/brand/Logo'

/** A closed study book turning slowly in 3D (six CSS faces). Decorative. */
export function Book3D() {
  return (
    <div className="tb-book-stage" aria-hidden>
      <div className="tb-book">
        <div className="tb-book-front">
          <span className="tb-book-ribbon" />
          <LogoMark className="size-9 [&_rect]:fill-[#fffcf6] [&_path]:stroke-[#b5643f]" />
          <div>
            <span className="block h-1.5 w-16 rounded-full bg-[#fffcf6]/80" />
            <span className="mt-2 block h-1.5 w-11 rounded-full bg-[#fffcf6]/45" />
            <p className="mt-4 font-display text-lg leading-none font-semibold tracking-[-0.02em]">telebeland</p>
          </div>
        </div>
        <div className="tb-book-back" />
        <div className="tb-book-spine" />
        <div className="tb-book-pages" />
        <div className="tb-book-top" />
        <div className="tb-book-bottom" />
      </div>
      <span className="tb-book-shadow" />
    </div>
  )
}
