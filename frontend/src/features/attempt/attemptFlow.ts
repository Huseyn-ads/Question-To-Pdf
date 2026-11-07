import { isTransient } from '@/api/errors'
import type { AnswerFeedback, SavedAnswer, SnapshotQuestion } from '@/api/types'

/** A final answer with the server's verdict. */
export interface CheckedAnswer extends AnswerFeedback {
  optionId: string
}

export type AnswerMap = Record<string, CheckedAnswer>

export function answersFromServer(answers: SavedAnswer[]): AnswerMap {
  const map: AnswerMap = {}
  for (const a of answers) {
    map[a.quiz_question_id] = {
      optionId: a.quiz_answer_option_id,
      is_correct: a.is_correct,
      correct_option_ids: a.correct_option_ids,
    }
  }
  return map
}

/**
 * Questions go strictly in order. The student starts (or resumes after a
 * reload) on the first question without an answer; when every question is
 * answered, the last one stays on screen with the finish action.
 */
export function resumeIndex(questions: SnapshotQuestion[], answers: AnswerMap): number {
  const index = questions.findIndex((q) => !answers[q.id])
  return index === -1 ? Math.max(questions.length - 1, 0) : index
}

/** The next unanswered question after `from`, or null when none is left. */
export function nextIndex(questions: SnapshotQuestion[], answers: AnswerMap, from: number): number | null {
  for (let i = from + 1; i < questions.length; i++) {
    if (!answers[questions[i]!.id]) return i
  }
  return null
}

export function tally(questions: SnapshotQuestion[], answers: AnswerMap) {
  let correct = 0
  let wrong = 0
  for (const q of questions) {
    const a = answers[q.id]
    if (!a) continue
    if (a.is_correct) correct++
    else wrong++
  }
  return { correct, wrong, answered: correct + wrong, unanswered: questions.length - correct - wrong }
}

/**
 * Sends one answer. Re-sending the same option is idempotent on the server,
 * so transient failures are retried a few times before the user sees an error.
 */
export async function sendWithRetry<T>(
  send: () => Promise<T>,
  { delays = [600, 1800], sleep = (ms: number) => new Promise<void>((r) => setTimeout(r, ms)) } = {},
): Promise<T> {
  for (let attempt = 0; ; attempt++) {
    try {
      return await send()
    } catch (error) {
      const delay = delays[attempt]
      if (delay === undefined || !isTransient(error)) throw error
      await sleep(delay)
    }
  }
}
