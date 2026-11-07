import { ApiError, isTransient } from '@/api/errors'

export type QuestionSaveStatus = 'saving' | 'saved' | 'error'

export interface QuestionSaveState {
  status: QuestionSaveStatus
  /** The option the user picked most recently (what the UI shows). */
  optionId: string
  error?: unknown
}

export interface SaverSnapshot {
  version: number
  questions: Readonly<Record<string, QuestionSaveState>>
  /** Aggregate for the status indicator. */
  overall: 'idle' | 'saving' | 'saved' | 'error'
  failed: string[]
}

export interface FlushResult {
  failed: string[]
}

type Send = (questionId: string, optionId: string) => Promise<void>

interface Entry {
  desired: string
  confirmed: string | null
  inFlight: string | null
  retryTimer: ReturnType<typeof setTimeout> | null
  retries: number
  status: QuestionSaveStatus
  error?: unknown
}

export interface AnswerSaverOptions {
  /** Backoff delays for automatic retries of transient failures (PUT is idempotent). */
  retryDelays?: number[]
  /** Called once when the server reports the attempt is closed (409). */
  onClosed?: (error: ApiError) => void
}

/**
 * Serialises answer saves per question and always converges on the latest
 * choice: while one PUT is in flight, newer picks are queued and only the
 * last one is sent afterwards. `flush()` lets "finish" wait until every
 * choice has reached the server (or definitively failed).
 */
export class AnswerSaver {
  private entries = new Map<string, Entry>()
  private listeners = new Set<() => void>()
  private waiters: Array<(result: FlushResult) => void> = []
  private stopped = false
  private touched = false
  private snapshot: SaverSnapshot = { version: 0, questions: {}, overall: 'idle', failed: [] }
  private readonly send: Send
  private readonly retryDelays: number[]
  private readonly onClosed: AnswerSaverOptions['onClosed']

  constructor(send: Send, options: AnswerSaverOptions = {}) {
    this.send = send
    this.retryDelays = options.retryDelays ?? [800, 2500]
    this.onClosed = options.onClosed
  }

  subscribe = (listener: () => void) => {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  getSnapshot = () => this.snapshot

  /** Seeds already-saved answers from the server so re-picking them is a no-op. */
  seed(answers: Record<string, string>) {
    for (const [questionId, optionId] of Object.entries(answers)) {
      const entry = this.entries.get(questionId)
      if (!entry) {
        this.entries.set(questionId, {
          desired: optionId,
          confirmed: optionId,
          inFlight: null,
          retryTimer: null,
          retries: 0,
          status: 'saved',
        })
      } else if (entry.inFlight === null && entry.retryTimer === null && entry.status !== 'error') {
        // Server truth wins for questions without local pending work.
        entry.desired = optionId
        entry.confirmed = optionId
      }
    }
    this.emit()
  }

  select(questionId: string, optionId: string) {
    if (this.stopped) return
    const entry = this.entries.get(questionId) ?? {
      desired: optionId,
      confirmed: null,
      inFlight: null,
      retryTimer: null,
      retries: 0,
      status: 'saving' as const,
    }
    this.touched = true
    entry.desired = optionId
    entry.retries = 0
    if (entry.retryTimer) {
      clearTimeout(entry.retryTimer)
      entry.retryTimer = null
    }
    this.entries.set(questionId, entry)
    if (entry.desired === entry.confirmed && entry.inFlight === null) {
      entry.status = 'saved'
      this.emit()
      return
    }
    entry.status = 'saving'
    this.emit()
    this.pump(questionId)
  }

  retryFailed() {
    for (const [questionId, entry] of this.entries) {
      if (entry.status === 'error') {
        entry.retries = 0
        entry.status = 'saving'
        this.pump(questionId)
      }
    }
    this.emit()
  }

  hasPending() {
    for (const entry of this.entries.values()) {
      if (entry.inFlight !== null || entry.retryTimer !== null || entry.desired !== entry.confirmed) return true
    }
    return false
  }

  /** Resolves when no save is in flight or scheduled. */
  flush(): Promise<FlushResult> {
    if (this.isSettled()) return Promise.resolve({ failed: this.failedIds() })
    return new Promise((resolve) => this.waiters.push(resolve))
  }

  /** Stops sending (attempt closed or finished); pending waiters resolve. */
  stop() {
    this.stopped = true
    for (const entry of this.entries.values()) {
      if (entry.retryTimer) clearTimeout(entry.retryTimer)
      entry.retryTimer = null
    }
    this.emit()
  }

  private isSettled() {
    for (const entry of this.entries.values()) {
      if (entry.inFlight !== null || entry.retryTimer !== null) return false
      if (!this.stopped && entry.status === 'saving') return false
    }
    return true
  }

  private failedIds() {
    const failed: string[] = []
    for (const [questionId, entry] of this.entries) {
      if (entry.status === 'error' || (this.stopped && entry.desired !== entry.confirmed)) failed.push(questionId)
    }
    return failed
  }

  private pump(questionId: string) {
    const entry = this.entries.get(questionId)
    if (!entry || this.stopped || entry.inFlight !== null || entry.retryTimer !== null) return
    if (entry.desired === entry.confirmed) {
      entry.status = 'saved'
      this.emit()
      return
    }
    const optionId = entry.desired
    entry.inFlight = optionId
    this.send(questionId, optionId).then(
      () => {
        entry.inFlight = null
        entry.confirmed = optionId
        entry.error = undefined
        if (entry.desired !== entry.confirmed) {
          this.pump(questionId)
        } else {
          entry.status = 'saved'
        }
        this.emit()
      },
      (error: unknown) => {
        entry.inFlight = null
        if (error instanceof ApiError && error.kind === 'conflict') {
          entry.status = 'error'
          entry.error = error
          this.stop()
          this.onClosed?.(error)
          return
        }
        if (isTransient(error) && entry.retries < this.retryDelays.length && !this.stopped) {
          const delay = this.retryDelays[entry.retries] ?? 1000
          entry.retries += 1
          entry.retryTimer = setTimeout(() => {
            entry.retryTimer = null
            this.pump(questionId)
          }, delay)
          this.emit()
          return
        }
        entry.status = 'error'
        entry.error = error
        this.emit()
      },
    )
  }

  private emit() {
    const questions: Record<string, QuestionSaveState> = {}
    let saving = false
    let error = false
    let saved = false
    for (const [questionId, entry] of this.entries) {
      questions[questionId] = { status: entry.status, optionId: entry.desired, error: entry.error }
      if (entry.status === 'saving') saving = true
      else if (entry.status === 'error') error = true
      else if (entry.confirmed !== null) saved = true
    }
    this.snapshot = {
      version: this.snapshot.version + 1,
      questions,
      overall: error ? 'error' : saving ? 'saving' : saved && this.touched ? 'saved' : 'idle',
      failed: this.failedIds(),
    }
    for (const listener of this.listeners) listener()
    if (this.isSettled() && this.waiters.length) {
      const result = { failed: this.failedIds() }
      const waiters = this.waiters
      this.waiters = []
      for (const resolve of waiters) resolve(result)
    }
  }
}
