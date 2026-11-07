import { describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/api/errors'
import type { SnapshotQuestion } from '@/api/types'
import { answersFromServer, nextIndex, resumeIndex, sendWithRetry, tally, type AnswerMap } from './attemptFlow'

const questions: SnapshotQuestion[] = ['q1', 'q2', 'q3'].map((id, i) => ({
  id,
  text: id,
  position: i + 1,
  options: [
    { id: `${id}-a`, text: 'A', position: 1 },
    { id: `${id}-b`, text: 'B', position: 2 },
  ],
}))

const answer = (optionId: string, right: boolean) => ({
  optionId,
  is_correct: right,
  correct_option_ids: [right ? optionId : `${optionId}-other`],
})

describe('attempt order', () => {
  it('starts on the first question without an answer', () => {
    expect(resumeIndex(questions, {})).toBe(0)
    expect(resumeIndex(questions, { q1: answer('q1-a', true) })).toBe(1)
  })

  it('stays on the last question once everything is answered', () => {
    const all: AnswerMap = { q1: answer('q1-a', true), q2: answer('q2-a', false), q3: answer('q3-b', true) }
    expect(resumeIndex(questions, all)).toBe(2)
    expect(nextIndex(questions, all, 2)).toBeNull()
  })

  it('moves forward only, skipping questions answered elsewhere', () => {
    const answers: AnswerMap = { q1: answer('q1-a', true), q2: answer('q2-b', true) }
    expect(nextIndex(questions, answers, 0)).toBe(2)
    expect(nextIndex(questions, { q1: answer('q1-a', true) }, 0)).toBe(1)
  })

  it('counts right and wrong answers', () => {
    expect(tally(questions, { q1: answer('q1-a', true), q2: answer('q2-a', false) })).toEqual({
      correct: 1,
      wrong: 1,
      answered: 2,
      unanswered: 1,
    })
  })

  it('reads verdicts that come back with the attempt', () => {
    const map = answersFromServer([
      { id: 'x', quiz_question_id: 'q1', quiz_answer_option_id: 'q1-b', is_correct: false, correct_option_ids: ['q1-a'] },
    ])
    expect(map.q1).toEqual({ optionId: 'q1-b', is_correct: false, correct_option_ids: ['q1-a'] })
  })
})

describe('sendWithRetry', () => {
  const sleep = () => Promise.resolve()

  it('retries network failures of the same answer, then succeeds', async () => {
    const send = vi
      .fn<() => Promise<string>>()
      .mockRejectedValueOnce(new ApiError({ kind: 'network', status: 0 }))
      .mockResolvedValueOnce('ok')
    await expect(sendWithRetry(send, { sleep })).resolves.toBe('ok')
    expect(send).toHaveBeenCalledTimes(2)
  })

  it('gives up after the last retry', async () => {
    const send = vi.fn<() => Promise<string>>().mockRejectedValue(new ApiError({ kind: 'server', status: 502 }))
    await expect(sendWithRetry(send, { sleep })).rejects.toMatchObject({ kind: 'server' })
    expect(send).toHaveBeenCalledTimes(3)
  })

  it('never retries a locked or closed answer', async () => {
    const send = vi
      .fn<() => Promise<string>>()
      .mockRejectedValue(new ApiError({ kind: 'answer_locked', status: 409, serverMessage: 'question already answered' }))
    await expect(sendWithRetry(send, { sleep })).rejects.toMatchObject({ kind: 'answer_locked' })
    expect(send).toHaveBeenCalledTimes(1)
  })
})
