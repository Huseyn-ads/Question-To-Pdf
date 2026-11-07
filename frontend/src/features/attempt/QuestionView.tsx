import { forwardRef, type ReactNode } from 'react'
import { Check } from 'lucide-react'
import type { SnapshotQuestion } from '@/api/types'
import { useT } from '@/i18n'
import { cn } from '@/lib/utils'
import { optionLetter } from './optionLetter'
import { splitQuestionText } from './questionText'


interface QuestionViewProps {
  question: SnapshotQuestion
  number: number
  total: number
  selectedOptionId: string | undefined
  disabled: boolean
  onSelect: (optionId: string) => void
  status?: ReactNode
}

/**
 * One question with native radio inputs (arrow keys move between options,
 * Space selects). Text keeps the document's line breaks for code and
 * matching tables.
 */
export const QuestionView = forwardRef<HTMLHeadingElement, QuestionViewProps>(function QuestionView(
  { question, number, total, selectedOptionId, disabled, onSelect, status },
  headingRef,
) {
  const t = useT()
  const headingId = `q-${question.id}-title`
  const contextId = `q-${question.id}-context`
  const { prompt, context, contextIsCode } = splitQuestionText(question.text)
  return (
    <article aria-labelledby={headingId} aria-describedby={context ? contextId : undefined}>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
        <p className="text-sm font-semibold tracking-wide text-forest uppercase">
          {t.attempt.questionLabel(number)}
          <span className="font-normal tracking-normal text-ink-subtle normal-case"> / {total}</span>
        </p>
        {status}
      </div>
      <h2
        id={headingId}
        ref={headingRef}
        tabIndex={-1}
        className="font-display text-[1.1875rem] leading-[1.6] font-medium text-pretty whitespace-pre-wrap text-ink outline-none [overflow-wrap:anywhere] md:text-question md:leading-[1.55]"
      >
        {prompt}
      </h2>
      {context && (
        <div
          id={contextId}
          className={cn(
            'mt-4 whitespace-pre-wrap text-ink [overflow-wrap:anywhere]',
            contextIsCode
              ? 'overflow-x-auto rounded-md border border-line bg-sunken/60 px-4 py-3 whitespace-pre font-mono text-[0.9375rem] leading-relaxed [overflow-wrap:normal]'
              : 'text-reading',
          )}
        >
          {context}
        </div>
      )}

      <fieldset className="mt-7" disabled={disabled}>
        <legend className="sr-only">{t.attempt.optionsLabel}</legend>
        <div className="flex flex-col gap-2.5">
          {question.options.map((option, index) => {
            const checked = option.id === selectedOptionId
            return (
              <label
                key={option.id}
                className={cn(
                  'group relative flex cursor-pointer items-start gap-3.5 rounded-md border px-3.5 py-3 transition-[border-color,background-color] duration-150 sm:gap-4 sm:px-4 sm:py-3.5',
                  'has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-forest',
                  checked
                    ? 'border-forest bg-forest-tint'
                    : 'border-line bg-surface hover:border-control/70 hover:bg-[#FBF7EF]',
                  disabled && 'cursor-not-allowed opacity-70 hover:border-line hover:bg-surface',
                )}
              >
                <input
                  type="radio"
                  name={`question-${question.id}`}
                  value={option.id}
                  checked={checked}
                  onChange={() => onSelect(option.id)}
                  className="peer sr-only"
                />
                <span
                  aria-hidden
                  className={cn(
                    'mt-px grid size-8 shrink-0 place-items-center rounded-full border text-sm font-semibold transition-colors',
                    checked
                      ? 'border-forest bg-forest text-surface'
                      : 'border-control/60 bg-surface text-ink-muted group-hover:border-control',
                  )}
                >
                  {checked ? <Check className="size-4" strokeWidth={2.5} /> : optionLetter(index)}
                </span>
                <span className="min-w-0 flex-1 self-center text-reading whitespace-pre-wrap text-ink [overflow-wrap:anywhere]">
                  <span className="sr-only">{optionLetter(index)}. </span>
                  {option.text}
                </span>
              </label>
            )
          })}
        </div>
      </fieldset>
    </article>
  )
})
