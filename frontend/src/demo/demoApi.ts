/**
 * DEMO API — an in-memory stand-in for the Go backend with the same
 * interface as `api` (src/api/endpoints.ts). It exists only so the design can
 * be reviewed without a running backend. State lives in sessionStorage so a
 * reload restores the attempt exactly like the real server does.
 */
import { api as realApi, type Api } from '@/api/endpoints'
import { ApiError } from '@/api/errors'
import type {
  AttemptDetails,
  AttemptResult,
  AttemptStatus,
  AttemptSummary,
  QuizFull,
  ResultQuestion,
  SnapshotQuestion,
} from '@/api/types'
import { demoBanks, demoQuestionPool, demoQuizzes, demoUser } from './data'

export type DemoScenario = 'timed' | 'short' | 'unlimited'

export interface DemoSettings {
  failSaves: boolean
  slowSaves: boolean
  scenario: DemoScenario
  empty: boolean
}

export const defaultDemoSettings: DemoSettings = {
  failSaves: false,
  slowSaves: false,
  scenario: 'timed',
  empty: false,
}

interface StoredAttempt {
  id: string
  quiz_id: string
  status: AttemptStatus
  started_at: string
  deadline_at: string | null
  finished_at: string | null
  answers: Record<string, string>
}

interface DemoState {
  settings: DemoSettings
  attempts: StoredAttempt[]
  seq: number
}

const STORAGE_KEY = 'telebeland.demo.v1'

const minutes = (n: number) => n * 60_000
const iso = (ms: number) => new Date(ms).toISOString()

function quizQuestions(quizId: string, count: number): SnapshotQuestion[] {
  return demoQuestionPool.slice(0, count).map((q, i) => {
    const id = `${quizId}-q${i + 1}`
    return {
      id,
      text: q.text,
      position: i + 1,
      options: q.options.map((text, j) => ({ id: `${id}-o${j + 1}`, text, position: j + 1 })),
    }
  })
}

const optionId = (quizId: string, q: number, o: number) => `${quizId}-q${q}-o${o + 1}`

/** Same verdict the backend returns with every answer. */
function feedback(questionId: string, chosen: string) {
  const index = Number(questionId.slice(questionId.lastIndexOf('-q') + 2)) - 1
  const right = `${questionId}-o${(demoQuestionPool[index]?.correct ?? 0) + 1}`
  return { is_correct: chosen === right, correct_option_ids: [right] }
}

function seedState(settings: DemoSettings): DemoState {
  const now = Date.now()
  const algoStart = settings.scenario === 'short' ? now - minutes(24) : now - minutes(6)
  const algoDeadline =
    settings.scenario === 'unlimited' ? null : settings.scenario === 'short' ? now + 40_000 : algoStart + minutes(25)
  const pick = (quizId: string, q: number, right: boolean) => {
    const correct = demoQuestionPool[q - 1]?.correct ?? 0
    return optionId(quizId, q, right ? correct : (correct + 1) % 4)
  }
  return {
    settings,
    seq: 10,
    attempts: settings.empty
      ? []
      : [
          {
            id: 'att-algo-2',
            quiz_id: 'quiz-algo',
            status: 'in_progress',
            started_at: iso(algoStart),
            deadline_at: algoDeadline ? iso(algoDeadline) : null,
            finished_at: null,
            answers: {
              'quiz-algo-q1': pick('quiz-algo', 1, true),
              'quiz-algo-q2': pick('quiz-algo', 2, true),
              'quiz-algo-q3': pick('quiz-algo', 3, false),
              'quiz-algo-q4': pick('quiz-algo', 4, true),
            },
          },
          {
            id: 'att-phil-1',
            quiz_id: 'quiz-phil',
            status: 'completed',
            started_at: iso(now - 86_400_000 - minutes(40)),
            deadline_at: null,
            finished_at: iso(now - 86_400_000 - minutes(22)),
            answers: Object.fromEntries(
              Array.from({ length: 10 }, (_, i) => [`quiz-phil-q${i + 1}`, pick('quiz-phil', i + 1, i < 8)]),
            ),
          },
          {
            id: 'att-arch-1',
            quiz_id: 'quiz-arch',
            status: 'expired',
            started_at: iso(now - 4 * 86_400_000),
            deadline_at: iso(now - 4 * 86_400_000 + minutes(15)),
            finished_at: iso(now - 4 * 86_400_000 + minutes(15)),
            answers: Object.fromEntries(
              Array.from({ length: 10 }, (_, i) => [`quiz-arch-q${i + 1}`, pick('quiz-arch', i + 1, i < 7)]),
            ),
          },
        ],
  }
}

function load(): DemoState {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (raw) return JSON.parse(raw) as DemoState
  } catch {
    /* storage unavailable — fall through to a fresh seed */
  }
  return seedState(defaultDemoSettings)
}

let state: DemoState = load()

function persist() {
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  } catch {
    /* ignore: demo keeps working in memory */
  }
}

export function getDemoSettings(): DemoSettings {
  return state.settings
}

export function resetDemo(settings: DemoSettings) {
  state = seedState(settings)
  persist()
}

const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

function notFound(): never {
  throw new ApiError({ kind: 'not_found', status: 404, serverMessage: 'not found' })
}

function notInDemo(): never {
  throw new ApiError({ kind: 'forbidden', status: 403, serverMessage: 'not available in demo' })
}

function findQuiz(quizId: string) {
  if (state.settings.empty) notFound()
  return demoQuizzes().find((q) => q.id === quizId) ?? notFound()
}

function findAttempt(attemptId: string) {
  return state.attempts.find((a) => a.id === attemptId) ?? notFound()
}

function isPastDeadline(a: StoredAttempt, now = Date.now()) {
  return a.deadline_at !== null && now >= Date.parse(a.deadline_at)
}

/** Mirrors the backend: an overdue in-progress attempt is closed as expired on read. */
function settle(a: StoredAttempt) {
  if (a.status === 'in_progress' && isPastDeadline(a)) {
    a.status = 'expired'
    a.finished_at = a.deadline_at
    persist()
  }
}

function details(a: StoredAttempt): AttemptDetails {
  const quiz = findQuiz(a.quiz_id)
  return {
    id: a.id,
    quiz_id: a.quiz_id,
    status: a.status,
    started_at: a.started_at,
    deadline_at: a.deadline_at,
    finished_at: a.finished_at,
    server_time: new Date().toISOString(),
    questions: quizQuestions(quiz.id, quiz.question_count),
    answers: Object.entries(a.answers).map(([qid, oid]) => ({
      id: `ans-${qid}`,
      quiz_question_id: qid,
      quiz_answer_option_id: oid,
      ...feedback(qid, oid),
    })),
  }
}

function result(a: StoredAttempt): AttemptResult {
  const quiz = findQuiz(a.quiz_id)
  const correct: ResultQuestion[] = []
  const wrong: ResultQuestion[] = []
  const unanswered: ResultQuestion[] = []
  quizQuestions(quiz.id, quiz.question_count).forEach((q, i) => {
    const correctIndex = demoQuestionPool[i]?.correct ?? 0
    const selected = a.answers[q.id] ?? null
    const options = q.options.map((o, j) => ({ ...o, is_correct: j === correctIndex, is_selected: o.id === selected }))
    const isCorrect = selected === null ? null : options.some((o) => o.is_selected && o.is_correct)
    const item: ResultQuestion = { ...q, selected_option_id: selected, is_correct: isCorrect, options }
    if (isCorrect === null) unanswered.push(item)
    else if (isCorrect) correct.push(item)
    else wrong.push(item)
  })
  const total = quiz.question_count
  return {
    attempt_id: a.id,
    quiz_id: a.quiz_id,
    status: a.status === 'expired' ? 'expired' : 'completed',
    total_questions: total,
    answered_questions: correct.length + wrong.length,
    correct_answers: correct.length,
    wrong_answers: wrong.length,
    unanswered_questions: unanswered.length,
    percentage: total ? (correct.length / total) * 100 : 0,
    started_at: a.started_at,
    finished_at: a.finished_at ?? new Date().toISOString(),
    correct_answers_list: correct,
    wrong_answers_list: wrong,
    unanswered_questions_list: unanswered,
  }
}

function summary(a: StoredAttempt): AttemptSummary {
  settle(a)
  const r = a.status === 'in_progress' ? null : result(a)
  return {
    attempt_id: a.id,
    quiz_id: a.quiz_id,
    status: a.status,
    started_at: a.started_at,
    deadline_at: a.deadline_at,
    finished_at: a.finished_at,
    correct_answers: r?.correct_answers ?? 0,
    gradable_count: r?.total_questions ?? 0,
    percentage: r ? r.percentage : null,
  }
}

export const demoApi: Api = {
  auth: {
    ...realApi.auth,
    register: notInDemo,
    login: notInDemo,
    verifyEmail: notInDemo,
    resendVerification: notInDemo,
    logout: async () => {
      await wait(200)
    },
    me: async () => {
      await wait(120)
      return demoUser
    },
  },
  banks: {
    list: async () => {
      await wait(260)
      return state.settings.empty ? [] : demoBanks()
    },
    get: async (bankId) => {
      await wait(150)
      if (state.settings.empty) notFound()
      return demoBanks().find((b) => b.id === bankId) ?? notFound()
    },
    create: notInDemo,
    update: notInDemo,
    remove: notInDemo,
  },
  questions: { list: notInDemo, get: notInDemo, create: notInDemo, replace: notInDemo, remove: notInDemo },
  imports: { preview: notInDemo, confirm: notInDemo },
  quizzes: {
    create: notInDemo,
    list: async (params = {}) => {
      await wait(300)
      const limit = params.limit ?? 20
      const offset = params.offset ?? 0
      const items = state.settings.empty ? [] : demoQuizzes().slice(offset, offset + limit)
      return { items, limit, offset }
    },
    get: async (quizId) => {
      await wait(200)
      const quiz = findQuiz(quizId)
      const full: QuizFull = { ...quiz, questions: quizQuestions(quiz.id, quiz.question_count) }
      return full
    },
    remove: notInDemo,
  },
  attempts: {
    start: async (quizId) => {
      await wait(350)
      const quiz = findQuiz(quizId)
      const active = state.attempts.find((a) => a.quiz_id === quizId && (settle(a), a.status === 'in_progress'))
      if (active) return { attempt: details(active), created: false }
      const now = Date.now()
      state.seq += 1
      const attempt: StoredAttempt = {
        id: `att-${quizId.replace('quiz-', '')}-${state.seq}`,
        quiz_id: quizId,
        status: 'in_progress',
        started_at: iso(now),
        deadline_at: quiz.time_limit_minutes ? iso(now + minutes(quiz.time_limit_minutes)) : null,
        finished_at: null,
        answers: {},
      }
      state.attempts.unshift(attempt)
      persist()
      return { attempt: details(attempt), created: true }
    },
    history: async (quizId, params = {}) => {
      await wait(220)
      findQuiz(quizId)
      const limit = params.limit ?? 20
      const offset = params.offset ?? 0
      return state.attempts
        .filter((a) => a.quiz_id === quizId)
        .sort((a, b) => Date.parse(b.started_at) - Date.parse(a.started_at))
        .slice(offset, offset + limit)
        .map(summary)
    },
    get: async (attemptId) => {
      await wait(280)
      return details(findAttempt(attemptId))
    },
    saveAnswer: async (attemptId, quizQuestionId, optionIdValue) => {
      await wait(state.settings.slowSaves ? 2000 : 380)
      if (state.settings.failSaves) {
        throw new ApiError({ kind: 'network', status: 0 })
      }
      const a = findAttempt(attemptId)
      if (a.status !== 'in_progress' || isPastDeadline(a)) {
        throw new ApiError({ kind: 'conflict', status: 409, serverMessage: 'attempt is closed' })
      }
      const saved = a.answers[quizQuestionId]
      if (saved && saved !== optionIdValue) {
        throw new ApiError({ kind: 'answer_locked', status: 409, serverMessage: 'question already answered' })
      }
      a.answers[quizQuestionId] = optionIdValue
      persist()
      return {
        quiz_question_id: quizQuestionId,
        option_id: optionIdValue,
        updated_at: new Date().toISOString(),
        ...feedback(quizQuestionId, optionIdValue),
      }
    },
    finish: async (attemptId) => {
      await wait(500)
      const a = findAttempt(attemptId)
      if (a.status === 'in_progress') {
        const expired = isPastDeadline(a)
        a.status = expired ? 'expired' : 'completed'
        a.finished_at = expired ? a.deadline_at : new Date().toISOString()
        persist()
      }
      return result(a)
    },
    result: async (attemptId) => {
      await wait(300)
      const a = findAttempt(attemptId)
      settle(a)
      if (a.status === 'in_progress') {
        throw new ApiError({ kind: 'conflict', status: 409, serverMessage: 'attempt is still in progress' })
      }
      return result(a)
    },
  },
}
