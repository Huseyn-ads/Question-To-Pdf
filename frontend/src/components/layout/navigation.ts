import { BookOpenText, ClipboardList, FileUp, LayoutDashboard, Presentation, Sparkles, type LucideIcon } from 'lucide-react'
import type { Messages } from '@/i18n'
import type { Paths } from '@/app/paths'

export interface NavItem {
  key: string
  label: string
  shortLabel?: string
  icon: LucideIcon
  to: string
  end?: boolean
}

export interface ToolItem {
  key: string
  title: string
  shortTitle: string
  description: string
  icon: LucideIcon
  /** null = not available yet ("Скоро") */
  to: string | null
}

export function mainNav(t: Messages, paths: Paths): NavItem[] {
  return [
    { key: 'dashboard', label: t.nav.dashboard, icon: LayoutDashboard, to: paths.dashboard, end: true },
    { key: 'banks', label: t.nav.banks, shortLabel: t.nav.banksShort, icon: BookOpenText, to: paths.banks },
    { key: 'quizzes', label: t.nav.quizzes, icon: ClipboardList, to: paths.quizzes },
  ]
}

/**
 * Tool registry. Future generators (topic → quiz, topic → slides) are listed
 * now and switch on by giving them a route once their API exists.
 */
export function tools(t: Messages, paths: Paths): ToolItem[] {
  return [
    { key: 'import', icon: FileUp, to: paths.importDocx, ...t.tools.importDocx },
    { key: 'quiz-from-topic', icon: Sparkles, to: null, ...t.tools.quizFromTopic },
    { key: 'slides-from-topic', icon: Presentation, to: null, ...t.tools.slidesFromTopic },
  ]
}
