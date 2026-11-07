import { useFieldArray, useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { AlertCircle, Plus, Trash2 } from 'lucide-react'
import { z } from 'zod'
import { useApi } from '@/api/context'
import { isApiError } from '@/api/errors'
import { queryKeys } from '@/api/queryKeys'
import type { AuthorQuestion, UUID } from '@/api/types'
import { useT, type Messages } from '@/i18n'
import { cn } from '@/lib/utils'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { fieldControlClass } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Field, FieldError } from '@/components/common/forms'
import { errorMessage } from '@/components/common/states'
import { useToast } from '@/components/common/toast'
import { optionLetter } from '@/features/attempt/optionLetter'
import { checkQuestion, cleanQuestion } from './questionRules'

const MAX_OPTIONS = 12

function questionSchema(t: Messages) {
  return z
    .object({
      text: z.string(),
      options: z.array(z.object({ text: z.string() })),
      correct: z.number().int(),
    })
    .superRefine((value, ctx) => {
      const check = checkQuestion(
        value.text,
        value.options.map((option, index) => ({ text: option.text, is_correct: index === value.correct })),
      )
      if (check.question.includes('text_required')) {
        ctx.addIssue({ code: 'custom', path: ['text'], message: t.questionForm.textRequired })
      }
      if (check.question.includes('options_min')) {
        ctx.addIssue({ code: 'custom', path: ['options'], message: t.questionForm.optionsMin })
      }
      if (check.question.includes('correct_required')) {
        ctx.addIssue({ code: 'custom', path: ['correct'], message: t.questionForm.correctRequired })
      }
      for (const [index, problem] of Object.entries(check.options)) {
        ctx.addIssue({
          code: 'custom',
          path: ['options', Number(index), 'text'],
          message: problem === 'empty' ? t.questionForm.optionRequired : t.questionForm.optionsDuplicate,
        })
      }
    })
}
type QuestionValues = z.infer<ReturnType<typeof questionSchema>>

function toValues(question: AuthorQuestion | undefined): QuestionValues {
  if (!question) return { text: '', options: [{ text: '' }, { text: '' }, { text: '' }, { text: '' }], correct: -1 }
  return {
    text: question.text,
    options: question.options.map((option) => ({ text: option.text })),
    correct: question.options.findIndex((option) => option.is_correct),
  }
}

/** Create a question or fully replace one (PUT). Never retried automatically. */
export function QuestionFormDialog({
  open,
  onOpenChange,
  bankId,
  question,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  bankId: UUID
  question?: AuthorQuestion
}) {
  const t = useT()
  const api = useApi()
  const toast = useToast()
  const queryClient = useQueryClient()
  const form = useForm<QuestionValues>({
    resolver: zodResolver(questionSchema(t)),
    values: toValues(question),
  })
  const options = useFieldArray({ control: form.control, name: 'options' })
  const correct = useWatch({ control: form.control, name: 'correct' })

  const save = useMutation({
    mutationFn: (values: QuestionValues) => {
      const payload = cleanQuestion(
        values.text,
        values.options.map((option, index) => ({ text: option.text, is_correct: index === values.correct })),
      )
      return question ? api.questions.replace(bankId, question.id, payload) : api.questions.create(bankId, payload)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.questions(bankId) })
      toast.success(question ? t.banks.questionSaved : t.banks.questionCreated)
      close()
    },
  })

  const errors = form.formState.errors
  function close() {
    form.reset()
    save.reset()
    onOpenChange(false)
  }
  const uncertain = !question && isApiError(save.error, 'network')

  const removeOption = (index: number) => {
    options.remove(index)
    if (correct === index) form.setValue('correct', -1)
    else if (correct > index) form.setValue('correct', correct - 1)
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (save.isPending) return
        if (next) onOpenChange(true)
        else close()
      }}
    >
      <DialogContent className="max-w-2xl gap-6">
        <DialogHeader>
          <DialogTitle>{question ? t.questionForm.editTitle : t.questionForm.createTitle}</DialogTitle>
        </DialogHeader>
        <form
          id="question-form"
          noValidate
          className="flex flex-col gap-6"
          onSubmit={form.handleSubmit((values) => {
            if (!save.isPending) save.mutate(values)
          })}
        >
          {save.isError && (
            <Alert tone={uncertain ? 'warning' : 'danger'}>
              <AlertCircle aria-hidden />
              <AlertDescription>
                {uncertain ? t.questionForm.saveNoRetry : errorMessage(t, save.error)}
              </AlertDescription>
            </Alert>
          )}
          <Field label={t.questionForm.text} hint={t.questionForm.textHint} error={errors.text?.message}>
            {(control) => <Textarea {...control} rows={4} {...form.register('text')} />}
          </Field>

          <fieldset aria-describedby="question-options-hint">
            <legend className="text-sm font-medium text-ink">{t.questionForm.options}</legend>
            <p id="question-options-hint" className="mt-1 text-sm text-ink-subtle">
              {t.questionForm.optionsHint}
            </p>
            <ol className="mt-3 flex flex-col gap-2.5">
              {options.fields.map((field, index) => {
                const letter = optionLetter(index)
                const optionError = errors.options?.[index]?.text?.message
                const isCorrect = correct === index
                return (
                  <li key={field.id}>
                    <div className="flex items-center gap-2">
                      <label
                        className={cn(
                          'grid size-11 shrink-0 cursor-pointer place-items-center rounded-md border text-sm font-semibold',
                          'has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-forest',
                          isCorrect ? 'border-forest bg-forest text-surface' : 'border-control/70 bg-surface text-ink-muted',
                        )}
                        title={t.questionForm.markCorrect(letter)}
                      >
                        <input
                          type="radio"
                          name="correct"
                          className="sr-only"
                          checked={isCorrect}
                          onChange={() => form.setValue('correct', index, { shouldValidate: form.formState.isSubmitted })}
                          aria-label={t.questionForm.markCorrect(letter)}
                        />
                        {letter}
                      </label>
                      <input
                        className={cn(fieldControlClass, 'h-11')}
                        aria-label={t.questionForm.option(letter)}
                        aria-invalid={optionError ? true : undefined}
                        {...form.register(`options.${index}.text`)}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        onClick={() => removeOption(index)}
                        disabled={options.fields.length <= 2}
                        aria-label={t.questionForm.removeOption(letter)}
                      >
                        <Trash2 aria-hidden />
                      </Button>
                    </div>
                    {isCorrect && (
                      <p className="mt-1 ml-[3.25rem] text-xs font-medium text-forest">{t.banks.correct}</p>
                    )}
                    {optionError && <FieldError className="mt-1 ml-[3.25rem]">{optionError}</FieldError>}
                  </li>
                )
              })}
            </ol>
            {errors.options?.message && <FieldError className="mt-2">{errors.options.message}</FieldError>}
            {errors.correct?.message && <FieldError className="mt-2">{errors.correct.message}</FieldError>}
            {options.fields.length < MAX_OPTIONS && (
              <Button type="button" variant="ghost" className="mt-2" onClick={() => options.append({ text: '' })}>
                <Plus aria-hidden />
                {t.questionForm.addOption}
              </Button>
            )}
          </fieldset>
        </form>
        <DialogFooter>
          <Button variant="secondary" onClick={close} disabled={save.isPending}>
            {t.common.cancel}
          </Button>
          <Button type="submit" form="question-form" loading={save.isPending}>
            {question ? t.common.saveChanges : t.common.add}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
