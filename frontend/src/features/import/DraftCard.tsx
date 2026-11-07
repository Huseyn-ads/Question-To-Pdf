import { memo, useState, type Dispatch } from 'react'
import { AlertTriangle, Check, Pencil, Plus, RotateCcw, Trash2, X } from 'lucide-react'
import { useI18n } from '@/i18n'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { fieldControlClass } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { FieldError } from '@/components/common/forms'
import { QuestionText } from '@/components/common/QuestionText'
import { checkDraft, issueText, type Draft, type DraftAction } from './draftState'

/**
 * One recognised question. Picking the correct option is always one click
 * (the most common fix); text edits open on demand to keep long lists calm.
 */
export const DraftCard = memo(function DraftCard({
  draft,
  number,
  dispatch,
}: {
  draft: Draft
  number: number
  dispatch: Dispatch<DraftAction>
}) {
  const { t } = useI18n()
  const check = checkDraft(draft)
  const textProblem = check.question.includes('text_required') || Object.keys(check.options).length > 0
  // Opens by itself only on mount, so the editor never collapses mid-typing.
  const [editing, setEditing] = useState(textProblem)
  const showEditor = editing
  const invalid = !draft.removed && !check.valid
  const radioName = `correct-${draft.key}`

  const problems: string[] = []
  if (check.question.includes('text_required')) problems.push(t.questionForm.textRequired)
  if (check.question.includes('options_min')) problems.push(t.questionForm.optionsMin)
  if (check.question.includes('correct_required')) problems.push(t.questionForm.correctRequired)

  return (
    <li
      className={cn(
        'rounded-lg border bg-surface p-4 sm:p-5',
        draft.removed ? 'border-dashed border-line bg-paper' : invalid ? 'border-danger/45' : 'border-line',
      )}
      aria-label={`${t.attempt.questionLabel(number)}${invalid ? ` — ${t.importer.filterProblems}` : ''}`}
    >
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <span className="tabular text-sm font-semibold text-ink">{number}.</span>
        <span className="text-xs text-ink-subtle">{t.importer.sourceNumber(draft.source_number)}</span>
        {draft.removed && (
          <span className="rounded-sm bg-sunken px-2 py-0.5 text-xs font-medium text-ink-muted">{t.importer.removed}</span>
        )}
        <div className="ml-auto flex gap-1">
          {!draft.removed && (
            <Button variant="ghost" size="sm" onClick={() => setEditing((v) => !v)} aria-pressed={editing}>
              {editing ? <X aria-hidden /> : <Pencil aria-hidden />}
              {editing ? t.common.close : t.common.edit}
            </Button>
          )}
          <Button
            variant={draft.removed ? 'soft' : 'ghost'}
            size="sm"
            onClick={() => dispatch({ type: 'toggleRemoved', key: draft.key })}
          >
            {draft.removed ? <RotateCcw aria-hidden /> : <Trash2 aria-hidden />}
            {draft.removed ? t.importer.restoreQuestion : t.importer.removeQuestion}
          </Button>
        </div>
      </div>

      {draft.removed ? (
        <p className="mt-2 line-clamp-2 text-[0.9375rem] text-ink-subtle [overflow-wrap:anywhere]">{draft.text}</p>
      ) : (
        <>
          {(draft.issues.length > 0 || problems.length > 0) && (
            <ul className="mt-3 flex flex-col gap-1.5">
              {problems.map((problem) => (
                <li key={problem} className="flex items-start gap-2 text-sm font-medium text-danger">
                  <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden />
                  {problem}
                </li>
              ))}
              {draft.issues.map((issue, index) => (
                <li key={`${issue.code}-${index}`} className="flex items-start gap-2 text-sm text-warning-ink">
                  <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden />
                  {issueText(t, issue)}
                </li>
              ))}
            </ul>
          )}

          <div className="mt-3">
            {showEditor ? (
              <>
                <label htmlFor={`text-${draft.key}`} className="sr-only">
                  {t.questionForm.text}
                </label>
                <Textarea
                  id={`text-${draft.key}`}
                  value={draft.text}
                  rows={Math.min(8, Math.max(2, draft.text.split('\n').length + 1))}
                  aria-invalid={check.question.includes('text_required') || undefined}
                  onChange={(event) => dispatch({ type: 'setText', key: draft.key, text: event.target.value })}
                />
              </>
            ) : (
              <QuestionText text={draft.text} />
            )}
          </div>

          <fieldset className="mt-3">
            <legend className="sr-only">{t.questionForm.optionsHint}</legend>
            <ul className="grid gap-1.5">
              {draft.options.map((option) => {
                const problem = check.options[draft.options.indexOf(option)]
                return (
                  <li key={option.key}>
                    <div
                      className={cn(
                        'flex items-start gap-2.5 rounded-md border px-2.5 py-2',
                        option.is_correct ? 'border-forest/40 bg-forest-tint' : 'border-transparent',
                      )}
                    >
                      <label
                        className={cn(
                          'grid size-7 shrink-0 cursor-pointer place-items-center rounded-sm text-xs font-semibold',
                          'has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-forest',
                          option.is_correct ? 'bg-forest text-on-forest' : 'border border-control/70 bg-surface text-ink-muted',
                        )}
                      >
                        <input
                          type="radio"
                          name={radioName}
                          className="sr-only"
                          checked={option.is_correct}
                          onChange={() => dispatch({ type: 'setCorrect', key: draft.key, optionKey: option.key })}
                          aria-label={t.questionForm.markCorrect(option.label)}
                        />
                        {option.is_correct ? <Check className="size-4" aria-hidden /> : option.label}
                      </label>
                      {showEditor ? (
                        <input
                          value={option.text}
                          aria-label={t.questionForm.option(option.label)}
                          aria-invalid={problem ? true : undefined}
                          onChange={(event) =>
                            dispatch({
                              type: 'setOptionText',
                              key: draft.key,
                              optionKey: option.key,
                              text: event.target.value,
                            })
                          }
                          className={cn(fieldControlClass, 'h-9 px-2.5 text-[0.9375rem]')}
                        />
                      ) : (
                        <span className="min-w-0 flex-1 pt-0.5 text-[0.9375rem] text-ink [overflow-wrap:anywhere]">
                          {option.text}
                        </span>
                      )}
                      {option.is_correct && !showEditor && (
                        <span className="shrink-0 pt-1 text-xs font-medium text-forest">{t.banks.correct}</span>
                      )}
                      {showEditor && (
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => dispatch({ type: 'removeOption', key: draft.key, optionKey: option.key })}
                          aria-label={t.questionForm.removeOption(option.label)}
                        >
                          <Trash2 aria-hidden />
                        </Button>
                      )}
                    </div>
                    {problem && (
                      <FieldError className="mt-1 ml-12">
                        {problem === 'empty' ? t.questionForm.optionRequired : t.questionForm.optionsDuplicate}
                      </FieldError>
                    )}
                  </li>
                )
              })}
            </ul>
            {showEditor && (
              <Button variant="ghost" size="sm" className="mt-2" onClick={() => dispatch({ type: 'addOption', key: draft.key })}>
                <Plus aria-hidden />
                {t.questionForm.addOption}
              </Button>
            )}
          </fieldset>
        </>
      )}
    </li>
  )
})
