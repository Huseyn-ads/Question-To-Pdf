import { describe, expect, it } from 'vitest'
import type { ImportPreview } from '@/api/types'
import { draftReducer, initDrafts, summarize, toImportPayload } from './draftState'

const preview: ImportPreview = {
  questions_count: 2,
  can_import: true,
  issues: [
    { code: 'source_numbers_renumbered', message: 'renumbered' },
    { source_number: 2, code: 'correct_answer_missing', message: 'no answer' },
    { source_number: 9, code: 'content_skipped', message: 'skipped' },
  ],
  questions: [
    {
      source_number: 1,
      text: ' Столица Франции? ',
      options: [
        { label: 'A', text: 'Париж ', is_correct: true },
        { label: 'B', text: 'Лион', is_correct: false },
      ],
    },
    {
      source_number: 2,
      text: 'Bakı hansı dənizin sahilindədir?',
      options: [
        { label: 'A', text: 'Xəzər', is_correct: false },
        { label: 'B', text: 'Qara', is_correct: false },
      ],
    },
  ],
}

describe('import drafts', () => {
  it('attaches issues to questions and keeps document-level ones separate', () => {
    const state = initDrafts(preview)
    expect(state.drafts[1]?.issues.map((i) => i.code)).toEqual(['correct_answer_missing'])
    // No source_number, or a source number that has no question → document level.
    expect(state.documentIssues.map((i) => i.code)).toEqual(['source_numbers_renumbered', 'content_skipped'])
  })

  it('counts invalid questions and fixes them through edits', () => {
    let state = initDrafts(preview)
    expect(summarize(state.drafts)).toMatchObject({ included: 2, invalid: 1, removed: 0 })

    const second = state.drafts[1]!
    state = draftReducer(state, { type: 'setCorrect', key: second.key, optionKey: second.options[0]!.key })
    expect(summarize(state.drafts).invalid).toBe(0)
    expect(state.drafts[1]!.options.map((o) => o.is_correct)).toEqual([true, false])
  })

  it('excludes removed questions from the summary and the payload', () => {
    let state = initDrafts(preview)
    state = draftReducer(state, { type: 'toggleRemoved', key: state.drafts[1]!.key })
    expect(summarize(state.drafts)).toMatchObject({ included: 1, removed: 1, invalid: 0 })
    expect(toImportPayload(state.drafts)).toEqual([
      {
        source_number: 1,
        text: 'Столица Франции?',
        options: [
          { label: 'A', text: 'Париж', is_correct: true },
          { label: 'B', text: 'Лион', is_correct: false },
        ],
      },
    ])
  })

  it('gives an added option the next free label', () => {
    let state = initDrafts(preview)
    const first = state.drafts[0]!
    state = draftReducer(state, { type: 'addOption', key: first.key })
    expect(state.drafts[0]!.options.map((o) => o.label)).toEqual(['A', 'B', 'C'])
    // The new option is empty, so the question needs attention until filled or removed.
    expect(summarize(state.drafts).invalidKeys.has(first.key)).toBe(true)
  })
})
