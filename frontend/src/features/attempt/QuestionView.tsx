import { forwardRef } from 'react'
import { Check, LoaderCircle, X } from 'lucide-react'
import type { SnapshotQuestion } from '@/api/types'
import { useT } from '@/i18n'
import { cn } from '@/lib/utils'
import type { CheckedAnswer } from './attemptFlow'
import { optionLetter } from './optionLetter'
import { splitQuestionText } from './questionText'

interface QuestionViewProps {
  question: SnapshotQuestion
  number: number
  total: number
  /** Verdict once the question is answered. */
  answer: CheckedAnswer | undefined
  /** Option being sent right now. */
  sendingId: string | undefined
  disabled: boolean
  onChoose: (optionId: string) => void
}

type OptionState = 'idle' | 'sending' | 'chosen-right' | 'chosen-wrong' | 'right' | 'rest'

/**
 * One question. Options are buttons, not radios: a choice is final, so arrow
 * keys must never pick an answer by accident. Digits 1–9 work too (page level).
 */
export const QuestionView = forwardRef<HTMLHeadingElement, QuestionViewProps>(function QuestionView(
  { question, number, total, answer, sendingId, disabled, onChoose },
  headingRef,
) {
  const t = useT()
  const headingId = `q-${question.id}-title`
  const contextId = `q-${question.id}-context`
  const { prompt, context, contextIsCode } = splitQuestionText(question.text)
  const locked = disabled || Boolean(answer) || Boolean(sendingId)

  const stateOf = (optionId: string): OptionState => {
    if (!answer) return sendingId === optionId ? 'sending' : 'idle'
    const isRight = answer.correct_option_ids.includes(optionId)
    if (answer.optionId === optionId) return isRight ? 'chosen-right' : 'chosen-wrong'
    return isRight ? 'right' : 'rest'
  }

  return (
    <article aria-labelledby={headingId} aria-describedby={context ? contextId : undefined}>
      <p className="mb-4 text-sm font-semibold tracking-wide text-forest uppercase">
        {t.attempt.questionLabel(number)}
        <span className="font-normal tracking-normal text-ink-subtle normal-case"> / {total}</span>
      </p>
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

      <ul className="mt-7 flex flex-col gap-2.5" aria-label={t.attempt.optionsLabel}>
        {question.options.map((option, index) => {
          const state = stateOf(option.id)
          const letter = optionLetter(index)
          const verdict =
            state === 'chosen-right'
              ? t.attempt.feedback.yourRight
              : state === 'chosen-wrong'
                ? t.attempt.feedback.yourWrong
                : state === 'right'
                  ? t.attempt.feedback.rightOption
                  : null
          return (
            <li key={option.id}>
              <button
                type="button"
                disabled={locked}
                aria-disabled={locked || undefined}
                onClick={() => onChoose(option.id)}
                className={cn(
                  'group relative flex w-full items-start gap-3.5 rounded-md border px-3.5 py-3 text-left transition-[border-color,background-color,opacity] duration-200 sm:gap-4 sm:px-4 sm:py-3.5',
                  state === 'idle' &&
                    'border-line bg-surface enabled:cursor-pointer enabled:hover:border-control/70 enabled:hover:bg-hover disabled:opacity-70',
                  state === 'sending' && 'border-forest bg-forest-tint',
                  (state === 'chosen-right' || state === 'right') && 'border-forest bg-forest-tint',
                  state === 'chosen-right' && 'ring-1 ring-forest',
                  state === 'chosen-wrong' && 'border-terracotta bg-terracotta-soft ring-1 ring-terracotta',
                  state === 'rest' && 'border-line-soft bg-surface opacity-60',
                )}
              >
                <span
                  aria-hidden
                  className={cn(
                    'mt-px grid size-8 shrink-0 place-items-center rounded-full border text-sm font-semibold transition-colors',
                    state === 'idle' && 'border-control/60 bg-surface text-ink-muted group-enabled:group-hover:border-control',
                    state === 'sending' && 'border-forest bg-forest text-on-forest',
                    (state === 'chosen-right' || state === 'right') && 'border-forest bg-forest text-on-forest',
                    state === 'chosen-wrong' && 'border-terracotta-ink bg-terracotta-ink text-on-forest',
                    state === 'rest' && 'border-line bg-surface text-ink-subtle',
                  )}
                >
                  {state === 'sending' ? (
                    <LoaderCircle className="size-4 animate-spin" />
                  ) : state === 'chosen-wrong' ? (
                    <X className="size-4" strokeWidth={2.5} />
                  ) : state === 'chosen-right' || state === 'right' ? (
                    <Check className="size-4" strokeWidth={2.5} />
                  ) : (
                    letter
                  )}
                </span>
                <span className="min-w-0 flex-1 self-center">
                  <span className="block text-reading whitespace-pre-wrap text-ink [overflow-wrap:anywhere]">
                    <span className="sr-only">{letter}. </span>
                    {option.text}
                  </span>
                  {verdict && (
                    <span
                      className={cn(
                        'mt-1 block text-sm font-medium',
                        state === 'chosen-wrong' ? 'text-terracotta-ink' : 'text-forest',
                      )}
                    >
                      {verdict}
                    </span>
                  )}
                </span>
              </button>
            </li>
          )
        })}
      </ul>
    </article>
  )
})
