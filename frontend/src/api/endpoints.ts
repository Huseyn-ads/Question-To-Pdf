import { clearCsrfToken, request } from './http'
import type {
  AnswerMode,
  AttemptDetails,
  AttemptResult,
  AttemptSummary,
  AuthorQuestion,
  BankInput,
  BankPatch,
  ImportPreview,
  ImportQuestionDraft,
  ImportResult,
  LoginInput,
  MessageResponse,
  Page,
  PageParams,
  QuestionBank,
  QuestionInput,
  QuizCreateInput,
  QuizCreated,
  QuizFull,
  QuizSummary,
  RawQuizFull,
  RegisterInput,
  RegisterResponse,
  ResultQuestion,
  SaveAnswerResult,
  SnapshotQuestion,
  StartAttemptResult,
  User,
  UUID,
} from './types'

const byPosition = <T extends { position: number }>(a: T, b: T) => a.position - b.position

const enc = encodeURIComponent

// ── Normalisers: keep wire quirks out of the UI ────────────────────────────
export function normalizeQuiz(raw: RawQuizFull): QuizFull {
  const { Questions, ...meta } = raw
  return {
    ...meta,
    questions: (Questions ?? [])
      .map<SnapshotQuestion>((q) => ({
        id: q.id,
        text: q.text,
        position: q.position,
        options: (q.quiz_answer_options ?? [])
          .map((o) => ({ id: o.id, text: o.text, position: o.position }))
          .sort(byPosition),
      }))
      .sort(byPosition),
  }
}

export function normalizeAttempt(raw: AttemptDetails): AttemptDetails {
  return {
    ...raw,
    questions: (raw.questions ?? [])
      .map((q) => ({ ...q, options: [...(q.options ?? [])].sort(byPosition) }))
      .sort(byPosition),
    answers: raw.answers ?? [],
  }
}

function normalizeResultQuestions(list: ResultQuestion[] | null | undefined): ResultQuestion[] {
  return (list ?? [])
    .map((q) => ({
      ...q,
      // Drop the always-empty options[].quiz_question_id the backend sends.
      options: (q.options ?? [])
        .map(({ id, text, position, is_correct, is_selected }) => ({ id, text, position, is_correct, is_selected }))
        .sort(byPosition),
    }))
    .sort(byPosition)
}

export function normalizeResult(raw: AttemptResult): AttemptResult {
  return {
    ...raw,
    correct_answers_list: normalizeResultQuestions(raw.correct_answers_list),
    wrong_answers_list: normalizeResultQuestions(raw.wrong_answers_list),
    unanswered_questions_list: normalizeResultQuestions(raw.unanswered_questions_list),
  }
}

// ── Endpoints ───────────────────────────────────────────────────────────────
export const api = {
  auth: {
    register: async (input: RegisterInput) =>
      (await request<RegisterResponse>('/v1/auth/register', { method: 'POST', json: input })).data,
    login: async (input: LoginInput) => {
      clearCsrfToken()
      return (await request<User>('/v1/auth/login', { method: 'POST', json: input, context: 'login' })).data
    },
    verifyEmail: async (token: string) =>
      (await request<MessageResponse>('/v1/auth/verify-email', { method: 'POST', json: { token } })).data,
    resendVerification: async (email: string) =>
      (await request<MessageResponse>('/v1/auth/resend-verification', { method: 'POST', json: { email } })).data,
    me: async (signal?: AbortSignal) => (await request<User>('/v1/auth/me', { signal })).data,
    logout: async () => {
      try {
        await request<void>('/v1/auth/logout', { method: 'POST', csrf: true })
      } finally {
        clearCsrfToken()
      }
    },
    /** Browser navigation target — never fetched. */
    googleStartUrl: '/v1/auth/google/start',
  },

  banks: {
    list: async (signal?: AbortSignal) => (await request<QuestionBank[]>('/v1/question-banks', { signal })).data ?? [],
    get: async (bankId: UUID, signal?: AbortSignal) =>
      (await request<QuestionBank>(`/v1/question-banks/${enc(bankId)}`, { signal })).data,
    create: async (input: BankInput) =>
      (await request<QuestionBank>('/v1/question-banks', { method: 'POST', json: input, csrf: true })).data,
    update: async (bankId: UUID, patch: BankPatch) =>
      (await request<QuestionBank>(`/v1/question-banks/${enc(bankId)}`, { method: 'PATCH', json: patch, csrf: true }))
        .data,
    remove: async (bankId: UUID) => {
      await request<void>(`/v1/question-banks/${enc(bankId)}`, { method: 'DELETE', csrf: true })
    },
  },

  questions: {
    list: async (bankId: UUID, signal?: AbortSignal) =>
      ((await request<AuthorQuestion[]>(`/v1/question-banks/${enc(bankId)}/questions`, { signal })).data ?? []).map(
        (q) => ({ ...q, options: [...(q.options ?? [])].sort(byPosition) }),
      ),
    get: async (bankId: UUID, questionId: UUID, signal?: AbortSignal) =>
      (await request<AuthorQuestion>(`/v1/question-banks/${enc(bankId)}/questions/${enc(questionId)}`, { signal }))
        .data,
    create: async (bankId: UUID, input: QuestionInput) =>
      (
        await request<AuthorQuestion>(`/v1/question-banks/${enc(bankId)}/questions`, {
          method: 'POST',
          json: input,
          csrf: true,
        })
      ).data,
    replace: async (bankId: UUID, questionId: UUID, input: QuestionInput) =>
      (
        await request<AuthorQuestion>(`/v1/question-banks/${enc(bankId)}/questions/${enc(questionId)}`, {
          method: 'PUT',
          json: input,
          csrf: true,
        })
      ).data,
    remove: async (bankId: UUID, questionId: UUID) => {
      await request<void>(`/v1/question-banks/${enc(bankId)}/questions/${enc(questionId)}`, {
        method: 'DELETE',
        csrf: true,
      })
    },
  },

  imports: {
    preview: async (bankId: UUID, file: File, answerMode: AnswerMode = 'auto') => {
      const form = new FormData()
      form.set('file', file)
      if (answerMode !== 'auto') form.set('answer_mode', answerMode)
      return (
        await request<ImportPreview>(`/v1/question-banks/${enc(bankId)}/imports/preview`, {
          method: 'POST',
          form,
          csrf: true,
        })
      ).data
    },
    confirm: async (bankId: UUID, questions: ImportQuestionDraft[]) =>
      (
        await request<ImportResult>(`/v1/question-banks/${enc(bankId)}/imports`, {
          method: 'POST',
          json: { questions },
          csrf: true,
        })
      ).data,
  },

  quizzes: {
    create: async (bankId: UUID, input: QuizCreateInput) =>
      (
        await request<QuizCreated>(`/v1/question-banks/${enc(bankId)}/quizzes`, {
          method: 'POST',
          json: input,
          csrf: true,
        })
      ).data,
    list: async (params: PageParams = {}, signal?: AbortSignal) => {
      const { data } = await request<Page<QuizSummary>>('/v1/quizzes', {
        query: { limit: params.limit, offset: params.offset },
        signal,
      })
      return { ...data, items: data.items ?? [] }
    },
    get: async (quizId: UUID, signal?: AbortSignal) =>
      normalizeQuiz((await request<RawQuizFull>(`/v1/quizzes/${enc(quizId)}`, { signal })).data),
    remove: async (quizId: UUID) => {
      await request<void>(`/v1/quizzes/${enc(quizId)}`, { method: 'DELETE', csrf: true })
    },
  },

  attempts: {
    /** Starts a new attempt or returns the active one (200). */
    start: async (quizId: UUID): Promise<StartAttemptResult> => {
      const { status, data } = await request<AttemptDetails>(`/v1/quizzes/${enc(quizId)}/attempts`, {
        method: 'POST',
        csrf: true,
      })
      return { attempt: normalizeAttempt(data), created: status === 201 }
    },
    history: async (quizId: UUID, params: PageParams = {}, signal?: AbortSignal) =>
      (
        await request<AttemptSummary[]>(`/v1/quizzes/${enc(quizId)}/attempts`, {
          query: { limit: params.limit, offset: params.offset },
          signal,
        })
      ).data ?? [],
    get: async (attemptId: UUID, signal?: AbortSignal) =>
      normalizeAttempt((await request<AttemptDetails>(`/v1/attempts/${enc(attemptId)}`, { signal })).data),
    saveAnswer: async (attemptId: UUID, quizQuestionId: UUID, optionId: UUID, signal?: AbortSignal) =>
      (
        await request<SaveAnswerResult>(`/v1/attempts/${enc(attemptId)}/answers/${enc(quizQuestionId)}`, {
          method: 'PUT',
          json: { option_id: optionId },
          csrf: true,
          signal,
        })
      ).data,
    finish: async (attemptId: UUID) =>
      normalizeResult(
        (await request<AttemptResult>(`/v1/attempts/${enc(attemptId)}/finish`, { method: 'POST', csrf: true })).data,
      ),
    result: async (attemptId: UUID, signal?: AbortSignal) =>
      normalizeResult((await request<AttemptResult>(`/v1/attempts/${enc(attemptId)}/result`, { signal })).data),
  },
}

export type Api = typeof api
