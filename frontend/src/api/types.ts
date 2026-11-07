/**
 * Types mirror the Go response structs (internal/<package>/payload.go) and
 * docs/api.md. Raw* types describe wire shapes that the API layer normalises
 * before the UI sees them.
 */

export type UUID = string
/** RFC 3339 timestamp, e.g. 2026-11-02T12:00:00Z */
export type ISODateTime = string

// ── Auth ────────────────────────────────────────────────────────────────────
export interface User {
  id: UUID
  email: string
  name: string
  created_at: ISODateTime
  email_verified_at: ISODateTime | null
}

export interface RegisterInput {
  email: string
  name: string
  password: string
}

export interface RegisterResponse extends User {
  verification_required: boolean
  verification_email_sent: boolean
}

export interface LoginInput {
  email: string
  password: string
}

export interface MessageResponse {
  message: string
}

// ── Question banks & questions ──────────────────────────────────────────────
export interface QuestionBank {
  id: UUID
  title: string
  description: string
  created_at: ISODateTime
  updated_at: ISODateTime
}

export interface BankInput {
  title: string
  description: string
}

export type BankPatch = Partial<BankInput>

export interface AuthorOption {
  id: UUID
  text: string
  position: number
  is_correct: boolean
}

export interface AuthorQuestion {
  id: UUID
  question_bank_id: UUID
  text: string
  options: AuthorOption[]
  created_at: ISODateTime
  updated_at: ISODateTime
}

export interface QuestionInput {
  text: string
  options: { text: string; is_correct: boolean }[]
}

// ── DOCX import ─────────────────────────────────────────────────────────────
export type AnswerMode = 'auto' | 'all_a'

export interface ImportOptionDraft {
  label: string
  text: string
  is_correct: boolean
}

export interface ImportQuestionDraft {
  source_number: number
  text: string
  options: ImportOptionDraft[]
}

export interface ImportIssue {
  /** Absent when the issue concerns the whole document. */
  source_number?: number
  field?: string
  code: string
  message: string
}

export interface ImportPreview {
  questions_count: number
  can_import: boolean
  issues: ImportIssue[]
  questions: ImportQuestionDraft[]
}

export interface ImportResult {
  imported_questions: number
  imported_options: number
}

// ── Quizzes ─────────────────────────────────────────────────────────────────
export interface QuizCreateInput {
  question_count: number
  /** 0 = no timer, 1…180 minutes */
  time_limit_minutes: number
  shuffle_options: boolean
}

/** POST /question-banks/{id}/quizzes returns only these fields. */
export interface QuizCreated {
  id: UUID
  question_bank_id: UUID
  question_count: number
  created_at: ISODateTime
}

export interface QuizSummary extends QuizCreated {
  time_limit_minutes: number
  shuffle_options: boolean
}

export interface Page<T> {
  items: T[]
  limit: number
  offset: number
}

export interface PageParams {
  limit?: number
  offset?: number
}

/** Snapshot option without correctness (shared by quiz and attempt). */
export interface SnapshotOption {
  id: UUID
  text: string
  position: number
}

export interface SnapshotQuestion {
  /** quiz_question_id — used in PUT /attempts/{id}/answers/{quizQuestionID} */
  id: UUID
  text: string
  position: number
  options: SnapshotOption[]
}

export interface QuizFull extends QuizSummary {
  questions: SnapshotQuestion[]
}

/** Wire shape of GET /v1/quizzes/{id}: capitalised `Questions`, nested names differ. */
export interface RawQuizFull extends QuizSummary {
  Questions: {
    id: UUID
    /** Contains the quiz UUID, not a question UUID (see docs/api.md). */
    question_id: UUID
    text: string
    position: number
    quiz_answer_options: { id: UUID; quiz_question_id: UUID; text: string; position: number }[]
  }[]
}

// ── Attempts ────────────────────────────────────────────────────────────────
export type AttemptStatus = 'in_progress' | 'completed' | 'expired'

/**
 * Answers are final: the first choice locks the question, and the server
 * reveals right away whether it was correct.
 */
export interface AnswerFeedback {
  is_correct: boolean
  /** Every correct option of the question; empty when it has no answer key. */
  correct_option_ids: UUID[]
}

export interface SavedAnswer extends AnswerFeedback {
  id: UUID
  quiz_question_id: UUID
  quiz_answer_option_id: UUID
}

export interface AttemptDetails {
  id: UUID
  quiz_id: UUID
  status: AttemptStatus
  started_at: ISODateTime
  /** null = no timer */
  deadline_at: ISODateTime | null
  finished_at: ISODateTime | null
  server_time: ISODateTime
  questions: SnapshotQuestion[]
  answers: SavedAnswer[]
}

export interface StartAttemptResult {
  attempt: AttemptDetails
  /** true for 201 (new attempt), false for 200 (resumed active attempt) */
  created: boolean
}

export interface SaveAnswerResult extends AnswerFeedback {
  quiz_question_id: UUID
  option_id: UUID
  updated_at: ISODateTime
}

export interface ResultOption {
  id: UUID
  text: string
  position: number
  is_correct: boolean
  is_selected: boolean
}

export interface ResultQuestion {
  id: UUID
  text: string
  position: number
  selected_option_id: UUID | null
  /** null for an unanswered question */
  is_correct: boolean | null
  options: ResultOption[]
}

export interface AttemptResult {
  attempt_id: UUID
  quiz_id: UUID
  status: Exclude<AttemptStatus, 'in_progress'>
  total_questions: number
  answered_questions: number
  correct_answers: number
  wrong_answers: number
  unanswered_questions: number
  percentage: number
  started_at: ISODateTime
  finished_at: ISODateTime
  correct_answers_list: ResultQuestion[]
  wrong_answers_list: ResultQuestion[]
  unanswered_questions_list: ResultQuestion[]
}

export interface AttemptSummary {
  attempt_id: UUID
  quiz_id: UUID
  status: AttemptStatus
  started_at: ISODateTime
  deadline_at: ISODateTime | null
  finished_at: ISODateTime | null
  correct_answers: number
  gradable_count: number
  /** null while the attempt is in progress */
  percentage: number | null
}
