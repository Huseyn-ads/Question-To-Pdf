import { useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { AlertCircle, ArrowLeft, FileUp, Play } from 'lucide-react'
import { z } from 'zod'
import { useApi } from '@/api/context'
import { isApiError } from '@/api/errors'
import { queryKeys } from '@/api/queryKeys'
import type { QuizCreated } from '@/api/types'
import { usePaths } from '@/app/paths'
import { useI18n, type Messages } from '@/i18n'
import { cn } from '@/lib/utils'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input, fieldControlClass } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { CheckboxRow, Field, FieldError } from '@/components/common/forms'
import { EmptyState, ErrorState, PageContainer, PageHeader, Panel, errorMessage } from '@/components/common/states'
import { useBankQuestions, useBanks } from '@/features/banks/hooks'

const TIME_PRESETS = [0, 15, 30, 45, 60] as const
const MAX_MINUTES = 180

// The upper bound depends on the chosen bank and is checked on submit.
function quizSchema(t: Messages) {
  return z.object({
    bankId: z.string().min(1, t.quizzes.bankChoose),
    questionCount: z.number({ error: t.quizzes.countPositive }).int(t.quizzes.countPositive).min(1, t.quizzes.countPositive),
    timeLimit: z
      .number({ error: t.quizzes.timeInvalid })
      .int(t.quizzes.timeInvalid)
      .min(0, t.quizzes.timeInvalid)
      .max(MAX_MINUTES, t.quizzes.timeInvalid),
    shuffle: z.boolean(),
  })
}
type QuizValues = z.infer<ReturnType<typeof quizSchema>>

export function NewQuizPage() {
  const { t, plural } = useI18n()
  const api = useApi()
  const paths = usePaths()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [searchParams] = useSearchParams()
  const banks = useBanks()
  const [customTime, setCustomTime] = useState(false)

  const form = useForm<QuizValues>({
    resolver: zodResolver(quizSchema(t)),
    defaultValues: { bankId: searchParams.get('bank') ?? '', questionCount: 20, timeLimit: 30, shuffle: true },
  })
  const bankId = useWatch({ control: form.control, name: 'bankId' })
  const questions = useBankQuestions(bankId || undefined)
  const available = questions.data?.length ?? 0

  // Keep the count inside the chosen bank's size when the bank changes.
  useEffect(() => {
    if (!questions.data) return
    const current = form.getValues('questionCount')
    if (!current || current > available) form.setValue('questionCount', Math.min(20, available) || 1)
  }, [questions.data, available, form])

  // Creation is never retried on its own: a repeat makes another random quiz.
  const create = useMutation({
    mutationFn: (input: { values: QuizValues; start: boolean }) =>
      api.quizzes.create(input.values.bankId, {
        question_count: input.values.questionCount,
        time_limit_minutes: input.values.timeLimit,
        shuffle_options: input.values.shuffle,
      }),
    onSuccess: async (quiz: QuizCreated, input) => {
      void queryClient.invalidateQueries({ queryKey: ['quizzes', 'list'] })
      if (!input.start) {
        void navigate(paths.quiz(quiz.id), { replace: true })
        return
      }
      try {
        const { attempt } = await api.attempts.start(quiz.id)
        queryClient.setQueryData(queryKeys.attempt(attempt.id), attempt)
        void navigate(paths.attempt(attempt.id), { replace: true })
      } catch {
        // The quiz exists; let the user start it from its page.
        void navigate(paths.quiz(quiz.id), { replace: true })
      }
    },
  })

  const errors = form.formState.errors
  const uncertain = isApiError(create.error, 'network')
  const submit = (start: boolean) =>
    form.handleSubmit((values) => {
      if (values.questionCount > available) {
        form.setError('questionCount', { message: t.quizzes.countInvalid(Math.max(available, 1)) }, { shouldFocus: true })
        return
      }
      if (!create.isPending) create.mutate({ values, start })
    })

  if (banks.isError) {
    return (
      <PageContainer>
        <Panel>
          <ErrorState title={t.banks.loadError} error={banks.error} onRetry={() => void banks.refetch()} />
        </Panel>
      </PageContainer>
    )
  }

  return (
    <PageContainer className="pb-10">
      <Link
        to={paths.quizzes}
        className="inline-flex items-center gap-1.5 rounded-sm text-sm font-medium text-ink-muted hover:text-forest"
      >
        <ArrowLeft className="size-4" aria-hidden />
        {t.quizzes.title}
      </Link>
      <div className="mt-4">
        <PageHeader title={t.quizzes.newTitle} subtitle={t.quizzes.newSubtitle} />
      </div>

      {banks.data?.length === 0 ? (
        <Panel className="mt-8 max-w-2xl">
          <EmptyState
            title={t.quizzes.noBanksTitle}
            text={t.quizzes.noBanksText}
            action={
              <Button asChild>
                <Link to={paths.importDocx}>
                  <FileUp aria-hidden />
                  {t.dashboard.actionImport}
                </Link>
              </Button>
            }
          />
        </Panel>
      ) : (
        <Panel className="mt-8 max-w-2xl p-5 sm:p-7">
          <form noValidate className="flex flex-col gap-7" onSubmit={submit(false)}>
            {create.isError && (
              <Alert tone={uncertain ? 'warning' : 'danger'}>
                <AlertCircle aria-hidden />
                <AlertDescription>
                  <p>{uncertain ? t.quizzes.createUnknown : errorMessage(t, create.error)}</p>
                  {uncertain && (
                    <Button asChild variant="link">
                      <Link to={paths.quizzes}>{t.quizzes.openQuizzes}</Link>
                    </Button>
                  )}
                </AlertDescription>
              </Alert>
            )}

            <Field
              label={t.quizzes.bank}
              error={errors.bankId?.message}
              hint={
                bankId && questions.data
                  ? available === 0
                    ? t.quizzes.bankEmpty
                    : t.quizzes.bankQuestions(plural(available, t.quiz.questions))
                  : undefined
              }
            >
              {(control) =>
                banks.isPending ? (
                  <Skeleton className="h-11" />
                ) : (
                  <select {...control} className={cn(fieldControlClass, 'h-11 pr-8')} {...form.register('bankId')}>
                    <option value="" disabled>
                      {t.quizzes.bankChoose}
                    </option>
                    {banks.data.map((bank) => (
                      <option key={bank.id} value={bank.id}>
                        {bank.title}
                      </option>
                    ))}
                  </select>
                )
              }
            </Field>

            <Field
              label={t.quizzes.count}
              error={errors.questionCount?.message}
              hint={available > 0 ? t.quizzes.countRange(available) : undefined}
            >
              {(control) => (
                <div className="flex flex-wrap items-center gap-2.5">
                  <Input
                    {...control}
                    type="number"
                    inputMode="numeric"
                    min={1}
                    max={available || undefined}
                    className="w-28"
                    disabled={!bankId || available === 0}
                    {...form.register('questionCount', { valueAsNumber: true })}
                  />
                  {available > 0 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => form.setValue('questionCount', available, { shouldValidate: true })}
                    >
                      {t.quizzes.countAll(available)}
                    </Button>
                  )}
                </div>
              )}
            </Field>

            <Controller
              control={form.control}
              name="timeLimit"
              render={({ field, fieldState }) => (
                <fieldset>
                  <legend className="text-sm font-medium text-ink">{t.quizzes.time}</legend>
                  <div className="mt-2 flex flex-wrap gap-2" role="radiogroup" aria-label={t.quizzes.time}>
                    {TIME_PRESETS.map((minutes) => {
                      const active = !customTime && field.value === minutes
                      return (
                        <button
                          key={minutes}
                          type="button"
                          role="radio"
                          aria-checked={active}
                          onClick={() => {
                            setCustomTime(false)
                            field.onChange(minutes)
                          }}
                          className={cn(
                            'h-10 rounded-md border px-3.5 text-[0.9375rem] font-medium transition-colors',
                            active
                              ? 'border-forest bg-forest-soft text-forest'
                              : 'border-control/60 bg-surface text-ink-muted hover:border-control hover:text-ink',
                          )}
                        >
                          {minutes === 0 ? t.quizzes.timeNone : t.quiz.minutesShort(minutes)}
                        </button>
                      )
                    })}
                    <button
                      type="button"
                      role="radio"
                      aria-checked={customTime}
                      onClick={() => setCustomTime(true)}
                      className={cn(
                        'h-10 rounded-md border px-3.5 text-[0.9375rem] font-medium transition-colors',
                        customTime
                          ? 'border-forest bg-forest-soft text-forest'
                          : 'border-control/60 bg-surface text-ink-muted hover:border-control hover:text-ink',
                      )}
                    >
                      {t.quizzes.timeCustom}
                    </button>
                  </div>
                  {customTime && (
                    <label className="mt-3 flex items-center gap-2.5 text-[0.9375rem] text-ink-muted">
                      <Input
                        type="number"
                        inputMode="numeric"
                        min={1}
                        max={MAX_MINUTES}
                        className="w-24"
                        aria-label={t.quizzes.timeMinutes}
                        value={Number.isFinite(field.value) ? field.value : ''}
                        aria-invalid={fieldState.error ? true : undefined}
                        onChange={(event) => field.onChange(event.target.valueAsNumber)}
                      />
                      {t.quizzes.timeMinutes}
                    </label>
                  )}
                  {fieldState.error?.message && <FieldError className="mt-2">{fieldState.error.message}</FieldError>}
                </fieldset>
              )}
            />

            <Controller
              control={form.control}
              name="shuffle"
              render={({ field }) => (
                <CheckboxRow
                  checked={field.value}
                  onChange={field.onChange}
                  label={t.quizzes.shuffle}
                  hint={t.quizzes.shuffleHint}
                />
              )}
            />

            <div className="flex flex-col-reverse gap-2.5 sm:flex-row">
              <Button
                type="submit"
                size="lg"
                variant="secondary"
                loading={create.isPending && !create.variables?.start}
                disabled={create.isPending || available === 0}
              >
                {t.quizzes.submit}
              </Button>
              <Button
                type="button"
                size="lg"
                onClick={() => void submit(true)()}
                loading={create.isPending && create.variables?.start}
                disabled={create.isPending || available === 0}
              >
                <Play aria-hidden />
                {t.quizzes.submitAndStart}
              </Button>
            </div>
          </form>
        </Panel>
      )}
    </PageContainer>
  )
}
