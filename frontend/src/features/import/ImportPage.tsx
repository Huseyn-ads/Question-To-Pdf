import { useEffect, useId, useMemo, useReducer, useRef, useState, type DragEvent } from 'react'
import { Link, useBlocker, useSearchParams } from 'react-router'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { AlertCircle, AlertTriangle, CheckCircle2, ClipboardList, FileText, FileUp, Upload } from 'lucide-react'
import { useApi } from '@/api/context'
import { isApiError } from '@/api/errors'
import { queryKeys } from '@/api/queryKeys'
import type { ImportResult, QuestionBank } from '@/api/types'
import { usePaths } from '@/app/paths'
import { useI18n, type Messages } from '@/i18n'
import { cn } from '@/lib/utils'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input, fieldControlClass } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { CheckboxRow, Field, FieldError } from '@/components/common/forms'
import { EmptyState, PageContainer, PageHeader, Panel, errorMessage } from '@/components/common/states'
import { bankSchema } from '@/features/banks/schemas'
import { useBanks } from '@/features/banks/hooks'
import { DraftCard } from './DraftCard'
import { draftReducer, initDrafts, issueText, summarize, toImportPayload, type DraftState } from './draftState'

const MAX_FILE_BYTES = 10 * 1024 * 1024
const NEW_BANK = '__new__'

function fileProblem(t: Messages, file: File): string | null {
  if (!file.name.toLowerCase().endsWith('.docx')) return t.importer.fileWrongType
  if (file.size > MAX_FILE_BYTES) return t.importer.fileTooLarge
  return null
}

function previewErrorText(t: Messages, error: unknown) {
  if (isApiError(error, 'too_large')) return t.importer.errorTooLarge
  if (isApiError(error, 'unsupported')) return t.importer.errorUnsupported
  if (isApiError(error, 'unprocessable')) return t.importer.errorUnprocessable
  if (isApiError(error, 'rate_limited')) return t.importer.errorRateLimited
  return errorMessage(t, error)
}

type Step = 'file' | 'review' | 'done'

export function ImportPage() {
  const { t, plural } = useI18n()
  const api = useApi()
  const queryClient = useQueryClient()
  const [searchParams] = useSearchParams()
  const banks = useBanks()

  const [step, setStep] = useState<Step>('file')
  const [bankChoice, setBankChoice] = useState<string>(searchParams.get('bank') ?? '')
  const [newTitle, setNewTitle] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [allA, setAllA] = useState(false)
  const [fieldErrors, setFieldErrors] = useState<{ bank?: string; file?: string }>({})
  const [targetBank, setTargetBank] = useState<QuestionBank | null>(null)
  const [state, dispatch] = useReducer(draftReducer, { drafts: [], documentIssues: [] } satisfies DraftState)
  const [result, setResult] = useState<ImportResult | null>(null)

  const bankList = useMemo(() => banks.data ?? [], [banks.data])
  // Without banks the only choice is a new one.
  const effectiveChoice = banks.data && bankList.length === 0 ? NEW_BANK : bankChoice
  const chosenBank = bankList.find((bank) => bank.id === effectiveChoice) ?? null

  // Bank creation and import confirmation are never retried automatically.
  const createBank = useMutation({ mutationFn: (title: string) => api.banks.create({ title, description: '' }) })
  const preview = useMutation({
    mutationFn: (input: { bank: QuestionBank; file: File; allA: boolean }) =>
      api.imports.preview(input.bank.id, input.file, input.allA ? 'all_a' : 'auto'),
    onSuccess: (data, input) => {
      dispatch({ type: 'reset', state: initDrafts(data) })
      setTargetBank(input.bank)
      setStep('review')
      window.scrollTo({ top: 0 })
    },
  })
  const confirm = useMutation({
    mutationFn: () => api.imports.confirm(targetBank?.id ?? '', toImportPayload(state.drafts)),
    onSuccess: (data) => {
      setResult(data)
      setStep('done')
      window.scrollTo({ top: 0 })
      if (targetBank) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.questions(targetBank.id) })
        void queryClient.invalidateQueries({ queryKey: queryKeys.banks })
      }
    },
  })

  const busy = createBank.isPending || preview.isPending

  async function analyze() {
    if (busy) return
    const errors: typeof fieldErrors = {}
    const bankSchemaResult = effectiveChoice === NEW_BANK ? bankSchema(t).shape.title.safeParse(newTitle) : null
    if (!effectiveChoice) errors.bank = t.importer.bankChoose
    else if (bankSchemaResult && !bankSchemaResult.success) errors.bank = bankSchemaResult.error.issues[0]?.message
    if (!file) errors.file = t.importer.fileRequired
    else if (fileProblem(t, file)) errors.file = fileProblem(t, file) ?? undefined
    setFieldErrors(errors)
    if (errors.bank || errors.file || !file) return

    let bank = chosenBank
    if (effectiveChoice === NEW_BANK && bankSchemaResult?.success) {
      try {
        bank = await createBank.mutateAsync(bankSchemaResult.data)
      } catch {
        return
      }
      // From now on the new bank is a regular choice, so a failed preview does not create a second bank.
      queryClient.setQueryData<QuestionBank[]>(queryKeys.banks, (list) => (list ? [bank!, ...list] : [bank!]))
      void queryClient.invalidateQueries({ queryKey: queryKeys.banks })
      setBankChoice(bank.id)
      setNewTitle('')
    }
    if (bank) preview.mutate({ bank, file, allA })
  }

  function restart() {
    setStep('file')
    setFile(null)
    setResult(null)
    preview.reset()
    confirm.reset()
    dispatch({ type: 'reset', state: { drafts: [], documentIssues: [] } })
    if (targetBank) setBankChoice(targetBank.id)
  }

  const dirty = step === 'review' && state.drafts.length > 0 && !confirm.isSuccess

  return (
    <PageContainer className="pb-10">
      <PageHeader title={t.importer.title} subtitle={step === 'file' ? t.importer.subtitle : undefined} />
      <Stepper step={step} />

      {step === 'file' && (
        <Panel className="mt-6 max-w-3xl p-5 sm:p-7">
          <form
            noValidate
            className="flex flex-col gap-7"
            onSubmit={(event) => {
              event.preventDefault()
              void analyze()
            }}
          >
            {(createBank.isError || preview.isError) && (
              <Alert tone="danger">
                <AlertCircle aria-hidden />
                <AlertDescription>
                  {createBank.isError
                    ? `${t.importer.bankCreateFailed} ${errorMessage(t, createBank.error)}`
                    : previewErrorText(t, preview.error)}
                </AlertDescription>
              </Alert>
            )}

            <div className="grid gap-4 sm:grid-cols-2">
              <Field label={t.importer.bank} hint={t.importer.bankHint} error={fieldErrors.bank}>
                {(control) =>
                  banks.isPending ? (
                    <Skeleton className="h-11" />
                  ) : (
                    <select
                      {...control}
                      value={effectiveChoice}
                      onChange={(event) => setBankChoice(event.target.value)}
                      className={cn(fieldControlClass, 'h-11 pr-8')}
                      disabled={busy}
                    >
                      <option value="" disabled>
                        {t.importer.bankChoose}
                      </option>
                      {bankList.map((bank) => (
                        <option key={bank.id} value={bank.id}>
                          {bank.title}
                        </option>
                      ))}
                      <option value={NEW_BANK}>{t.importer.bankNew}</option>
                    </select>
                  )
                }
              </Field>
              {effectiveChoice === NEW_BANK && (
                <Field label={t.importer.bankNewTitle}>
                  {(control) => (
                    <Input
                      {...control}
                      value={newTitle}
                      onChange={(event) => setNewTitle(event.target.value)}
                      placeholder={t.banks.fieldTitlePlaceholder}
                      disabled={busy}
                    />
                  )}
                </Field>
              )}
            </div>

            <FileDrop
              file={file}
              error={fieldErrors.file}
              disabled={busy}
              onFile={(next) => {
                setFile(next)
                setFieldErrors((errors) => ({ ...errors, file: next ? (fileProblem(t, next) ?? undefined) : undefined }))
                preview.reset()
              }}
            />

            <CheckboxRow checked={allA} onChange={setAllA} label={t.importer.allA} hint={t.importer.allAHint} disabled={busy} />

            <div>
              <Button type="submit" size="lg" loading={busy}>
                <Upload aria-hidden />
                {busy ? t.importer.analyzing : t.importer.analyze}
              </Button>
            </div>
          </form>
        </Panel>
      )}

      {step === 'review' && targetBank && (
        <Review
          state={state}
          dispatch={dispatch}
          bank={targetBank}
          confirm={confirm}
          onBack={restart}
          plural={plural}
        />
      )}

      {step === 'done' && targetBank && result && (
        <Done bank={targetBank} result={result} onMore={restart} />
      )}

      <LeaveGuard when={dirty} />
    </PageContainer>
  )
}

function Stepper({ step }: { step: Step }) {
  const { t } = useI18n()
  const steps: { key: Step; label: string }[] = [
    { key: 'file', label: t.importer.stepFile },
    { key: 'review', label: t.importer.stepReview },
    { key: 'done', label: t.importer.stepDone },
  ]
  const current = steps.findIndex((s) => s.key === step)
  return (
    <ol className="mt-6 flex items-center gap-2 text-sm" aria-label={t.importer.title}>
      {steps.map((s, index) => (
        <li key={s.key} className="flex items-center gap-2" aria-current={index === current ? 'step' : undefined}>
          {index > 0 && <span className="h-px w-6 bg-line sm:w-10" aria-hidden />}
          <span
            className={cn(
              'grid size-6 place-items-center rounded-full text-xs font-semibold',
              index < current && 'bg-forest-soft text-forest',
              index === current && 'bg-forest text-surface',
              index > current && 'border border-line text-ink-subtle',
            )}
            aria-hidden
          >
            {index < current ? <CheckCircle2 className="size-4" /> : index + 1}
          </span>
          <span className={cn(index === current ? 'font-semibold text-ink' : 'text-ink-muted')}>{s.label}</span>
        </li>
      ))}
    </ol>
  )
}

function FileDrop({
  file,
  error,
  disabled,
  onFile,
}: {
  file: File | null
  error?: string
  disabled: boolean
  onFile: (file: File | null) => void
}) {
  const { t, formatNumber } = useI18n()
  const inputId = useId()
  const inputRef = useRef<HTMLInputElement>(null)
  const [over, setOver] = useState(false)

  const onDrop = (event: DragEvent) => {
    event.preventDefault()
    setOver(false)
    if (disabled) return
    const dropped = event.dataTransfer.files[0]
    if (dropped) onFile(dropped)
  }

  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-sm font-medium text-ink" id={`${inputId}-label`}>
        {t.importer.file}
      </span>
      <div
        onDragOver={(event) => {
          event.preventDefault()
          if (!disabled) setOver(true)
        }}
        onDragLeave={() => setOver(false)}
        onDrop={onDrop}
        className={cn(
          'relative flex flex-col items-center gap-3 rounded-lg border-2 border-dashed px-5 py-8 text-center transition-colors',
          'has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-forest',
          over ? 'border-forest bg-forest-tint' : error ? 'border-danger/50 bg-danger-soft/40' : 'border-control/50 bg-paper',
        )}
      >
        <input
          ref={inputRef}
          id={inputId}
          type="file"
          accept=".docx,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
          className="sr-only"
          disabled={disabled}
          aria-labelledby={`${inputId}-label`}
          aria-describedby={`${inputId}-hint${error ? ` ${inputId}-error` : ''}`}
          aria-invalid={error ? true : undefined}
          onChange={(event) => {
            onFile(event.target.files?.[0] ?? null)
            event.target.value = ''
          }}
        />
        {file ? (
          <>
            <span className="grid size-12 place-items-center rounded-full bg-forest-soft text-forest">
              <FileText className="size-6" aria-hidden />
            </span>
            <p className="max-w-full font-medium break-all text-ink">{file.name}</p>
            <p className="text-sm text-ink-subtle">
              {formatNumber(file.size / 1024 / 1024, { maximumFractionDigits: 1, minimumFractionDigits: 1 })} МБ
            </p>
            <Button type="button" variant="secondary" size="sm" disabled={disabled} onClick={() => inputRef.current?.click()}>
              {t.importer.fileChange}
            </Button>
          </>
        ) : (
          <>
            <span className="grid size-12 place-items-center rounded-full bg-surface text-forest ring-1 ring-line">
              <FileUp className="size-6" aria-hidden />
            </span>
            <p className="text-ink">
              <span className="hidden sm:inline">
                {t.importer.drop} {t.importer.dropOr}{' '}
              </span>
              <label
                htmlFor={inputId}
                className="cursor-pointer font-semibold text-forest underline decoration-forest/40 underline-offset-4 first-letter:uppercase hover:decoration-forest"
              >
                {t.importer.browse}
              </label>
            </p>
          </>
        )}
      </div>
      {error && <FieldError id={`${inputId}-error`}>{error}</FieldError>}
      <p id={`${inputId}-hint`} className="text-sm text-ink-subtle">
        {t.importer.fileLimits}
      </p>
    </div>
  )
}

type Filter = 'all' | 'problems'

function Review({
  state,
  dispatch,
  bank,
  confirm,
  onBack,
  plural,
}: {
  state: DraftState
  dispatch: Parameters<typeof DraftCard>[0]['dispatch']
  bank: QuestionBank
  confirm: ReturnType<typeof useMutation<ImportResult, Error, void>>
  onBack: () => void
  plural: ReturnType<typeof useI18n>['plural']
}) {
  const { t } = useI18n()
  const paths = usePaths()
  const [filter, setFilter] = useState<Filter>('all')
  const summary = summarize(state.drafts)
  const issueTotal = state.documentIssues.length + state.drafts.reduce((sum, d) => sum + d.issues.length, 0)
  const withProblems = state.drafts.filter((d) => d.issues.length > 0 || summary.invalidKeys.has(d.key))
  const shown = filter === 'all' ? state.drafts : withProblems
  const uncertain = isApiError(confirm.error, 'network')
  const canConfirm = summary.included > 0 && summary.invalid === 0

  if (state.drafts.length === 0) {
    return (
      <Panel className="mt-6">
        <EmptyState
          title={t.importer.emptyTitle}
          text={t.importer.emptyText}
          action={
            <Button onClick={onBack}>
              <FileUp aria-hidden />
              {t.importer.backToFile}
            </Button>
          }
        />
        {state.documentIssues.length > 0 && <DocumentIssues issues={state.documentIssues} />}
      </Panel>
    )
  }

  return (
    <div className="mt-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div className="min-w-0">
          <h2 className="font-display text-xl font-semibold text-ink">{t.importer.reviewTitle}</h2>
          <p className="mt-1 text-ink-muted [overflow-wrap:anywhere]">
            {t.importer.reviewSummary(plural(state.drafts.length, t.quiz.questions), bank.title)}
            {issueTotal > 0 && ` · ${plural(issueTotal, t.importer.issuesCount)}`}
          </p>
        </div>
        <div role="radiogroup" aria-label={t.importer.filterLabel} className="flex rounded-md border border-line bg-surface p-1">
          {(['all', 'problems'] as const).map((value) => (
            <button
              key={value}
              type="button"
              role="radio"
              aria-checked={filter === value}
              onClick={() => setFilter(value)}
              className={cn(
                'h-8 rounded-sm px-3 text-sm font-medium whitespace-nowrap',
                filter === value ? 'bg-forest-soft text-forest' : 'text-ink-muted hover:text-ink',
              )}
            >
              {value === 'all' ? t.importer.filterAll : t.importer.filterProblems}
              <span className="tabular ml-1.5 text-xs">
                {value === 'all' ? state.drafts.length : withProblems.length}
              </span>
            </button>
          ))}
        </div>
      </div>

      {state.documentIssues.length > 0 && (
        <Panel className="mt-5">
          <DocumentIssues issues={state.documentIssues} />
        </Panel>
      )}

      {shown.length === 0 ? (
        <p className="mt-6 rounded-lg border border-line bg-surface px-5 py-8 text-center text-ink-muted">
          {t.importer.noProblems}
        </p>
      ) : (
        <ol className="mt-5 flex flex-col gap-3">
          {shown.map((draft) => (
            <DraftCard key={draft.key} draft={draft} number={state.drafts.indexOf(draft) + 1} dispatch={dispatch} />
          ))}
        </ol>
      )}

      {/* Sticky action bar: above the phone bottom navigation, flush on larger screens. */}
      <div className="sticky bottom-20 z-20 mt-6 md:bottom-4">
        <div className="rounded-lg border border-line bg-surface/97 p-4 shadow-float backdrop-blur-sm sm:p-5">
          {confirm.isError && (
            <Alert tone={uncertain ? 'warning' : 'danger'} className="mb-4">
              <AlertCircle aria-hidden />
              {uncertain && <AlertTitle>{t.errors.network}</AlertTitle>}
              <AlertDescription>
                <p>{uncertain ? t.importer.confirmUnknown : errorMessage(t, confirm.error)}</p>
                {uncertain && (
                  <Button asChild variant="link">
                    <Link to={paths.bank(bank.id)} target="_blank" rel="noopener">
                      {t.importer.openBankToCheck}
                    </Link>
                  </Button>
                )}
              </AlertDescription>
            </Alert>
          )}
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <p className="flex items-start gap-2 text-sm" aria-live="polite">
              {summary.invalid > 0 ? (
                <>
                  <AlertTriangle className="mt-0.5 size-4 shrink-0 text-danger" aria-hidden />
                  <span className="text-ink">
                    <b className="font-semibold">{plural(summary.invalid, t.importer.invalidCount)}</b>
                    <span className="text-ink-muted"> — {t.importer.confirmBlocked}</span>
                  </span>
                </>
              ) : summary.included === 0 ? (
                <span className="text-ink-muted">{t.importer.nothingToImport}</span>
              ) : (
                <>
                  <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-forest" aria-hidden />
                  <span className="text-ink">
                    {t.importer.allValid}
                    {summary.removed > 0 && (
                      <span className="text-ink-muted"> · {plural(summary.removed, t.importer.removedCount)}</span>
                    )}
                  </span>
                </>
              )}
            </p>
            <div className="flex flex-col-reverse gap-2 sm:flex-row">
              <Button variant="secondary" onClick={onBack} disabled={confirm.isPending}>
                {t.importer.backToFile}
              </Button>
              <Button
                onClick={() => {
                  if (!confirm.isPending && canConfirm) confirm.mutate()
                }}
                disabled={!canConfirm}
                loading={confirm.isPending}
              >
                {uncertain
                  ? t.importer.sendAgain
                  : t.importer.confirm(plural(summary.included, t.quiz.questions))}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function DocumentIssues({ issues }: { issues: DraftState['documentIssues'] }) {
  const { t } = useI18n()
  return (
    <div className="px-5 py-4">
      <h3 className="text-sm font-semibold text-ink">{t.importer.documentIssues}</h3>
      <ul className="mt-2 flex flex-col gap-1.5">
        {issues.map((issue, index) => (
          <li key={`${issue.code}-${index}`} className="flex items-start gap-2 text-sm text-warning-ink">
            <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden />
            <span>
              {issueText(t, issue)}
              {issue.source_number ? ` (${t.importer.sourceNumber(issue.source_number)})` : ''}
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
}

function Done({ bank, result, onMore }: { bank: QuestionBank; result: ImportResult; onMore: () => void }) {
  const { t, plural } = useI18n()
  const paths = usePaths()
  const heading = useRef<HTMLHeadingElement>(null)
  useEffect(() => heading.current?.focus(), [])
  return (
    <Panel className="mt-6 max-w-3xl p-6 sm:p-8">
      <span className="grid size-12 place-items-center rounded-full bg-forest-soft text-forest">
        <CheckCircle2 className="size-6" aria-hidden />
      </span>
      <h2 ref={heading} tabIndex={-1} className="mt-4 font-display text-2xl font-semibold text-ink outline-none">
        {t.importer.doneTitle}
      </h2>
      <p className="mt-2 text-reading text-ink-muted [overflow-wrap:anywhere]">
        {t.importer.doneText(plural(result.imported_questions, t.quiz.questions), bank.title)}
      </p>
      <div className="mt-6 flex flex-col gap-2.5 sm:flex-row sm:flex-wrap">
        <Button asChild size="lg">
          <Link to={paths.newQuizFrom(bank.id)}>
            <ClipboardList aria-hidden />
            {t.importer.createQuiz}
          </Link>
        </Button>
        <Button asChild size="lg" variant="secondary">
          <Link to={paths.bank(bank.id)}>{t.importer.openBank}</Link>
        </Button>
        <Button size="lg" variant="ghost" onClick={onMore}>
          <FileUp aria-hidden />
          {t.importer.importMore}
        </Button>
      </div>
    </Panel>
  )
}

/** Confirms before leaving with recognised but unsaved questions (in-app navigation and tab close). */
function LeaveGuard({ when }: { when: boolean }) {
  const { t } = useI18n()
  const blocker = useBlocker(({ currentLocation, nextLocation }) => when && currentLocation.pathname !== nextLocation.pathname)

  useEffect(() => {
    if (!when) return
    const onBeforeUnload = (event: BeforeUnloadEvent) => event.preventDefault()
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [when])

  return (
    <Dialog open={blocker.state === 'blocked'} onOpenChange={(open) => !open && blocker.reset?.()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t.importer.leaveTitle}</DialogTitle>
          <DialogDescription>{t.importer.leaveText}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="secondary" onClick={() => blocker.reset?.()}>
            {t.common.cancel}
          </Button>
          <Button variant="danger" onClick={() => blocker.proceed?.()}>
            {t.importer.leaveConfirm}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
