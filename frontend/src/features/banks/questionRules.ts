/**
 * Client-side copy of the backend question rules (internal/question/service.go,
 * prepareQuestionData): trimmed text is required, at least two options, every
 * option non-empty, options unique case-insensitively, exactly one correct.
 *
 * The manual create/replace endpoints enforce these rules. POST /imports does
 * not validate drafts on the server, so the import screen relies on this check
 * to keep invalid questions out of the bank.
 */
export interface OptionLike {
  text: string
  is_correct: boolean
}

export type QuestionProblem = 'text_required' | 'options_min' | 'correct_required'
export type OptionProblem = 'empty' | 'duplicate'

export interface QuestionCheck {
  valid: boolean
  question: QuestionProblem[]
  /** Problems per option index. */
  options: Partial<Record<number, OptionProblem>>
}

const normalize = (value: string) => value.trim().toLowerCase()

export function checkQuestion(text: string, options: readonly OptionLike[]): QuestionCheck {
  const question: QuestionProblem[] = []
  const optionProblems: Partial<Record<number, OptionProblem>> = {}

  if (!text.trim()) question.push('text_required')
  if (options.length < 2) question.push('options_min')

  const seen = new Map<string, number>()
  options.forEach((option, index) => {
    const key = normalize(option.text)
    if (!key) {
      optionProblems[index] = 'empty'
      return
    }
    const first = seen.get(key)
    if (first === undefined) seen.set(key, index)
    else optionProblems[index] = 'duplicate'
  })

  const correct = options.filter((option) => option.is_correct).length
  if (correct !== 1) question.push('correct_required')

  return {
    valid: question.length === 0 && Object.keys(optionProblems).length === 0,
    question,
    options: optionProblems,
  }
}

/** Payload as the backend stores it: trimmed text and options. */
export function cleanQuestion<T extends OptionLike>(text: string, options: readonly T[]) {
  return {
    text: text.trim(),
    options: options.map((option) => ({ ...option, text: option.text.trim() })),
  }
}
