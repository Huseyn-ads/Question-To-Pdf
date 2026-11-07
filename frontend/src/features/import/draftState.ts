import type { ImportIssue, ImportPreview, ImportQuestionDraft } from '@/api/types'
import type { Messages } from '@/i18n'
import { optionLetter } from '@/features/attempt/optionLetter'
import { checkQuestion, cleanQuestion, type QuestionCheck } from '@/features/banks/questionRules'

export interface DraftOption {
  key: string
  label: string
  text: string
  is_correct: boolean
}

export interface Draft {
  key: string
  source_number: number
  text: string
  options: DraftOption[]
  /** Excluded from the import by the user. */
  removed: boolean
  /** Issues the parser reported for this question. */
  issues: ImportIssue[]
}

export interface DraftState {
  drafts: Draft[]
  /** Issues without source_number: they concern the whole document. */
  documentIssues: ImportIssue[]
}

let seq = 0
const nextKey = (prefix: string) => `${prefix}${++seq}`

export function initDrafts(preview: ImportPreview): DraftState {
  const bySource = new Map<number, ImportIssue[]>()
  const documentIssues: ImportIssue[] = []
  for (const issue of preview.issues ?? []) {
    if (issue.source_number === undefined || issue.source_number === 0) {
      documentIssues.push(issue)
      continue
    }
    const list = bySource.get(issue.source_number) ?? []
    list.push(issue)
    bySource.set(issue.source_number, list)
  }
  const known = new Set((preview.questions ?? []).map((q) => q.source_number))
  // An issue pointing at a question that was not returned still has to be visible.
  for (const [source, issues] of bySource) if (!known.has(source)) documentIssues.push(...issues)

  return {
    documentIssues,
    drafts: (preview.questions ?? []).map((question) => ({
      key: nextKey('q'),
      source_number: question.source_number,
      text: question.text,
      removed: false,
      issues: bySource.get(question.source_number) ?? [],
      options: (question.options ?? []).map((option) => ({ key: nextKey('o'), ...option })),
    })),
  }
}

export type DraftAction =
  | { type: 'reset'; state: DraftState }
  | { type: 'setText'; key: string; text: string }
  | { type: 'setOptionText'; key: string; optionKey: string; text: string }
  | { type: 'setCorrect'; key: string; optionKey: string }
  | { type: 'addOption'; key: string }
  | { type: 'removeOption'; key: string; optionKey: string }
  | { type: 'toggleRemoved'; key: string }

function updateDraft(state: DraftState, key: string, update: (draft: Draft) => Draft): DraftState {
  return { ...state, drafts: state.drafts.map((draft) => (draft.key === key ? update(draft) : draft)) }
}

function nextLabel(options: DraftOption[]) {
  const used = new Set(options.map((option) => option.label.toUpperCase()))
  for (let i = 0; i < 26; i++) {
    const letter = optionLetter(i)
    if (!used.has(letter)) return letter
  }
  return String(options.length + 1)
}

export function draftReducer(state: DraftState, action: DraftAction): DraftState {
  switch (action.type) {
    case 'reset':
      return action.state
    case 'setText':
      return updateDraft(state, action.key, (draft) => ({ ...draft, text: action.text }))
    case 'setOptionText':
      return updateDraft(state, action.key, (draft) => ({
        ...draft,
        options: draft.options.map((o) => (o.key === action.optionKey ? { ...o, text: action.text } : o)),
      }))
    case 'setCorrect':
      return updateDraft(state, action.key, (draft) => ({
        ...draft,
        options: draft.options.map((o) => ({ ...o, is_correct: o.key === action.optionKey })),
      }))
    case 'addOption':
      return updateDraft(state, action.key, (draft) => ({
        ...draft,
        options: [...draft.options, { key: nextKey('o'), label: nextLabel(draft.options), text: '', is_correct: false }],
      }))
    case 'removeOption':
      return updateDraft(state, action.key, (draft) => ({
        ...draft,
        options: draft.options.filter((o) => o.key !== action.optionKey),
      }))
    case 'toggleRemoved':
      return updateDraft(state, action.key, (draft) => ({ ...draft, removed: !draft.removed }))
  }
}

export const checkDraft = (draft: Draft): QuestionCheck => checkQuestion(draft.text, draft.options)

export interface DraftSummary {
  included: number
  removed: number
  invalid: number
  /** Keys of included drafts that fail validation. */
  invalidKeys: Set<string>
}

export function summarize(drafts: readonly Draft[]): DraftSummary {
  const invalidKeys = new Set<string>()
  let removed = 0
  for (const draft of drafts) {
    if (draft.removed) removed++
    else if (!checkDraft(draft).valid) invalidKeys.add(draft.key)
  }
  return { included: drafts.length - removed, removed, invalid: invalidKeys.size, invalidKeys }
}

/** Confirm payload: exactly the backend fields (unknown JSON fields are rejected). */
export function toImportPayload(drafts: readonly Draft[]): ImportQuestionDraft[] {
  return drafts
    .filter((draft) => !draft.removed)
    .map((draft) => {
      const clean = cleanQuestion(draft.text, draft.options)
      return {
        source_number: draft.source_number,
        text: clean.text,
        options: clean.options.map(({ label, text, is_correct }) => ({ label, text, is_correct })),
      }
    })
}

/** Localised parser issue; unknown codes fall back to the backend message. */
export function issueText(t: Messages, issue: ImportIssue) {
  return t.importer.issueCodes[issue.code] ?? issue.message
}
