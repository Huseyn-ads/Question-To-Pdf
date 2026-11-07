import { splitQuestionText } from '@/features/attempt/questionText'
import { cn } from '@/lib/utils'

/**
 * Compact rendering of a question for lists (bank, import review, result):
 * the first line as the prompt, imported context (code, statements) below it.
 */
export function QuestionText({ text, className, as: Heading = 'p' }: { text: string; className?: string; as?: 'p' | 'h3' }) {
  const { prompt, context, contextIsCode } = splitQuestionText(text)
  return (
    <div className={cn('min-w-0', className)}>
      <Heading className="font-display text-[1.0625rem] leading-snug font-semibold text-pretty text-ink [overflow-wrap:anywhere]">
        {prompt}
      </Heading>
      {context &&
        (contextIsCode ? (
          <pre className="mt-2 overflow-x-auto rounded-md bg-sunken/70 px-3 py-2 font-mono text-[0.8125rem] leading-relaxed whitespace-pre text-ink">
            {context}
          </pre>
        ) : (
          <p className="mt-1.5 text-[0.9375rem] leading-relaxed whitespace-pre-wrap text-ink-muted [overflow-wrap:anywhere]">
            {context}
          </p>
        ))}
    </div>
  )
}
