import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/api/errors'
import { AnswerSaver } from './answerSaver'

type Call = { questionId: string; optionId: string; resolve: () => void; reject: (e: unknown) => void }

function harness(options: ConstructorParameters<typeof AnswerSaver>[1] = {}) {
  const calls: Call[] = []
  const saver = new AnswerSaver(
    (questionId, optionId) =>
      new Promise<void>((resolve, reject) => {
        calls.push({ questionId, optionId, resolve, reject })
      }),
    { retryDelays: [100, 200], ...options },
  )
  return { saver, calls }
}

const flushMicrotasks = () => new Promise((r) => setTimeout(r, 0))

describe('AnswerSaver', () => {
  beforeEach(() => vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] }))
  afterEach(() => vi.useRealTimers())

  it('sends only the latest choice after the in-flight save completes', async () => {
    const { saver, calls } = harness()
    saver.select('q1', 'a')
    saver.select('q1', 'b')
    saver.select('q1', 'c')
    expect(calls.map((c) => c.optionId)).toEqual(['a'])
    expect(saver.getSnapshot().overall).toBe('saving')

    calls[0]!.resolve()
    await vi.advanceTimersByTimeAsync(0)
    expect(calls.map((c) => c.optionId)).toEqual(['a', 'c'])

    calls[1]!.resolve()
    await vi.advanceTimersByTimeAsync(0)
    expect(saver.getSnapshot().questions.q1).toMatchObject({ status: 'saved', optionId: 'c' })
    expect(saver.getSnapshot().overall).toBe('saved')
  })

  it('flush waits for pending saves so finish counts the last choice', async () => {
    const { saver, calls } = harness()
    saver.select('q1', 'a')
    saver.select('q2', 'x')
    let settled = false
    const flushed = saver.flush().then((r) => {
      settled = true
      return r
    })
    await vi.advanceTimersByTimeAsync(0)
    expect(settled).toBe(false)
    calls.forEach((c) => c.resolve())
    await expect(flushed).resolves.toEqual({ failed: [] })
  })

  it('retries transient failures automatically, then reports an error', async () => {
    const { saver, calls } = harness()
    saver.select('q1', 'a')
    calls[0]!.reject(new ApiError({ kind: 'network', status: 0 }))
    await vi.advanceTimersByTimeAsync(0)
    expect(saver.getSnapshot().overall).toBe('saving')
    await vi.advanceTimersByTimeAsync(100)
    calls[1]!.reject(new ApiError({ kind: 'server', status: 500 }))
    await vi.advanceTimersByTimeAsync(200)
    calls[2]!.reject(new ApiError({ kind: 'network', status: 0 }))
    await vi.advanceTimersByTimeAsync(0)
    expect(calls).toHaveLength(3)
    expect(saver.getSnapshot().overall).toBe('error')
    await expect(saver.flush()).resolves.toEqual({ failed: ['q1'] })

    saver.retryFailed()
    expect(calls).toHaveLength(4)
    calls[3]!.resolve()
    await vi.advanceTimersByTimeAsync(0)
    expect(saver.getSnapshot().overall).toBe('saved')
  })

  it('stops on 409 and reports the attempt as closed', async () => {
    const onClosed = vi.fn()
    const { saver, calls } = harness({ onClosed })
    saver.select('q1', 'a')
    calls[0]!.reject(new ApiError({ kind: 'conflict', status: 409 }))
    await vi.advanceTimersByTimeAsync(0)
    expect(onClosed).toHaveBeenCalledOnce()
    saver.select('q2', 'b')
    expect(calls).toHaveLength(1)
  })

  it('does not resend answers that the server already has', async () => {
    const { saver, calls } = harness()
    saver.seed({ q1: 'a' })
    saver.select('q1', 'a')
    expect(calls).toHaveLength(0)
    expect(saver.getSnapshot().overall).toBe('saved')
    await flushMicrotasks
  })

  it('does not retry validation errors', async () => {
    const { saver, calls } = harness()
    saver.select('q1', 'a')
    calls[0]!.reject(new ApiError({ kind: 'validation', status: 400 }))
    await vi.advanceTimersByTimeAsync(1000)
    expect(calls).toHaveLength(1)
    expect(saver.getSnapshot().failed).toEqual(['q1'])
  })
})
