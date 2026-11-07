import type { AttemptResult, ResultQuestion } from '@/api/types'

export type Verdict = 'correct' | 'wrong' | 'skipped'

export interface ReviewItem {
  question: ResultQuestion
  verdict: Verdict
}

export function reviewItems(result: AttemptResult): ReviewItem[] {
  return [
    ...result.correct_answers_list.map((question) => ({ question, verdict: 'correct' as const })),
    ...result.wrong_answers_list.map((question) => ({ question, verdict: 'wrong' as const })),
    ...result.unanswered_questions_list.map((question) => ({ question, verdict: 'skipped' as const })),
  ].sort((a, b) => a.question.position - b.question.position)
}
