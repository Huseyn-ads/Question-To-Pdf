import { useDeferredValue, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, Check, ClipboardList, FileUp, MoreHorizontal, Pencil, Plus, Search, Trash2 } from 'lucide-react'
import { useApi } from '@/api/context'
import { isApiError } from '@/api/errors'
import { queryKeys } from '@/api/queryKeys'
import type { AuthorQuestion } from '@/api/types'
import { usePaths } from '@/app/paths'
import { useI18n } from '@/i18n'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { ConfirmDialog } from '@/components/common/ConfirmDialog'
import { QuestionText } from '@/components/common/QuestionText'
import { EmptyState, ErrorState, PageContainer, PageHeader, Panel } from '@/components/common/states'
import { useToast } from '@/components/common/toast'
import { optionLetter } from '@/features/attempt/optionLetter'
import { BankFormDialog } from './BankFormDialog'
import { QuestionFormDialog } from './QuestionFormDialog'
import { useBank, useBankQuestions } from './hooks'

const PAGE = 50

export function BankPage() {
  const { bankId = '' } = useParams()
  const { t, plural } = useI18n()
  const api = useApi()
  const paths = usePaths()
  const toast = useToast()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const bank = useBank(bankId)
  const questions = useBankQuestions(bankId)

  const [editBankOpen, setEditBankOpen] = useState(false)
  const [deleteBankOpen, setDeleteBankOpen] = useState(false)
  const [questionDialog, setQuestionDialog] = useState<{ open: boolean; question?: AuthorQuestion }>({ open: false })
  const [deleting, setDeleting] = useState<AuthorQuestion | null>(null)
  const [search, setSearch] = useState('')
  const [limit, setLimit] = useState(PAGE)
  const query = useDeferredValue(search.trim().toLowerCase())

  const removeBank = useMutation({
    mutationFn: () => api.banks.remove(bankId),
    onSuccess: () => {
      toast.success(t.banks.deleted)
      queryClient.removeQueries({ queryKey: queryKeys.bank(bankId) })
      // Quizzes and attempts of this bank are gone too.
      void queryClient.invalidateQueries({ queryKey: queryKeys.banks })
      void queryClient.invalidateQueries({ queryKey: ['quizzes'] })
      void navigate(paths.banks, { replace: true })
    },
  })
  const removeQuestion = useMutation({
    mutationFn: (question: AuthorQuestion) => api.questions.remove(bankId, question.id),
    onSuccess: (_, question) => {
      queryClient.setQueryData<AuthorQuestion[]>(queryKeys.questions(bankId), (list) =>
        list?.filter((item) => item.id !== question.id),
      )
      toast.success(t.banks.questionDeleted)
      setDeleting(null)
    },
    onError: (error, question) => {
      // Already deleted elsewhere: the goal is reached.
      if (isApiError(error, 'not_found')) {
        queryClient.setQueryData<AuthorQuestion[]>(queryKeys.questions(bankId), (list) =>
          list?.filter((item) => item.id !== question.id),
        )
        setDeleting(null)
      }
    },
  })

  const filtered = useMemo(() => {
    const list = questions.data ?? []
    if (!query) return list
    return list.filter(
      (q) => q.text.toLowerCase().includes(query) || q.options.some((o) => o.text.toLowerCase().includes(query)),
    )
  }, [questions.data, query])

  if (bank.isError) {
    return (
      <PageContainer>
        <BackLink />
        <Panel className="mt-6">
          <ErrorState
            title={t.banks.bankLoadError}
            error={bank.error}
            onRetry={isApiError(bank.error, 'not_found') ? undefined : () => void bank.refetch()}
            retrying={bank.isFetching}
          />
        </Panel>
      </PageContainer>
    )
  }

  const count = questions.data?.length ?? 0
  const visible = filtered.slice(0, limit)

  return (
    <PageContainer>
      <BackLink />
      <div className="mt-4">
        <PageHeader
          title={bank.data ? bank.data.title : <Skeleton className="h-9 w-72" />}
          subtitle={bank.data?.description || undefined}
          actions={
            <>
              <Button asChild variant="secondary">
                <Link to={paths.importInto(bankId)}>
                  <FileUp aria-hidden />
                  {t.banks.importDocx}
                </Link>
              </Button>
              {count > 0 ? (
                <Button asChild>
                  <Link to={paths.newQuizFrom(bankId)}>
                    <ClipboardList aria-hidden />
                    {t.banks.createQuiz}
                  </Link>
                </Button>
              ) : (
                <Button disabled title={t.banks.createQuizDisabled}>
                  <ClipboardList aria-hidden />
                  {t.banks.createQuiz}
                </Button>
              )}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon" aria-label={t.common.actions} disabled={!bank.data}>
                    <MoreHorizontal aria-hidden />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onSelect={() => setEditBankOpen(true)}>
                    <Pencil aria-hidden />
                    {t.common.edit}
                  </DropdownMenuItem>
                  <DropdownMenuItem variant="destructive" onSelect={() => setDeleteBankOpen(true)}>
                    <Trash2 aria-hidden />
                    {t.common.delete}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          }
        />
      </div>

      <section aria-labelledby="questions-title" className="mt-10">
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <h2 id="questions-title" className="font-display text-xl font-semibold text-ink">
            {t.banks.questionsTitle}
            {questions.data && <span className="ml-2 font-sans text-base font-normal text-ink-subtle">{count}</span>}
          </h2>
          <div className="flex flex-col gap-2.5 sm:flex-row sm:items-center">
            {count > 5 && (
              <div className="relative sm:w-72">
                <Search
                  className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-ink-subtle"
                  aria-hidden
                />
                <Input
                  type="search"
                  value={search}
                  onChange={(event) => {
                    setSearch(event.target.value)
                    setLimit(PAGE)
                  }}
                  placeholder={t.banks.searchPlaceholder}
                  aria-label={t.banks.searchLabel}
                  className="h-10 pl-9"
                />
              </div>
            )}
            <Button variant="soft" onClick={() => setQuestionDialog({ open: true })}>
              <Plus aria-hidden />
              {t.banks.addQuestion}
            </Button>
          </div>
        </div>

        {questions.isPending ? (
          <div className="flex flex-col gap-3" aria-hidden>
            {[0, 1, 2].map((i) => (
              <Skeleton key={i} className="h-40 rounded-lg" />
            ))}
          </div>
        ) : questions.isError ? (
          <Panel>
            <ErrorState
              title={t.banks.questionsLoadError}
              error={questions.error}
              onRetry={() => void questions.refetch()}
              retrying={questions.isFetching}
            />
          </Panel>
        ) : count === 0 ? (
          <Panel>
            <EmptyState
              title={t.banks.questionsEmptyTitle}
              text={t.banks.questionsEmptyText}
              action={
                <>
                  <Button asChild>
                    <Link to={paths.importInto(bankId)}>
                      <FileUp aria-hidden />
                      {t.banks.importDocx}
                    </Link>
                  </Button>
                  <Button variant="secondary" onClick={() => setQuestionDialog({ open: true })}>
                    <Plus aria-hidden />
                    {t.banks.addQuestion}
                  </Button>
                </>
              }
            />
          </Panel>
        ) : filtered.length === 0 ? (
          <Panel>
            <EmptyState illustration={false} title={t.banks.searchEmpty} />
          </Panel>
        ) : (
          <>
            {query && (
              <p className="mb-3 text-sm text-ink-muted" aria-live="polite">
                {t.banks.shownOf(filtered.length, count)}
              </p>
            )}
            <ol className="flex flex-col gap-3">
              {visible.map((question) => (
                <QuestionCard
                  key={question.id}
                  number={(questions.data ?? []).indexOf(question) + 1}
                  question={question}
                  onEdit={() => setQuestionDialog({ open: true, question })}
                  onDelete={() => {
                    removeQuestion.reset()
                    setDeleting(question)
                  }}
                />
              ))}
            </ol>
            {filtered.length > visible.length && (
              <div className="mt-4 flex justify-center">
                <Button variant="secondary" onClick={() => setLimit((value) => value + PAGE)}>
                  {t.common.showMore} · {plural(filtered.length - visible.length, t.quiz.questions)}
                </Button>
              </div>
            )}
          </>
        )}
      </section>

      {bank.data && <BankFormDialog open={editBankOpen} onOpenChange={setEditBankOpen} bank={bank.data} />}
      <QuestionFormDialog
        open={questionDialog.open}
        onOpenChange={(open) => setQuestionDialog((state) => ({ ...state, open }))}
        bankId={bankId}
        question={questionDialog.question}
      />
      <ConfirmDialog
        open={deleteBankOpen}
        onOpenChange={(open) => {
          setDeleteBankOpen(open)
          if (!open) removeBank.reset()
        }}
        title={t.banks.deleteTitle}
        text={t.banks.deleteText(bank.data?.title ?? '')}
        confirmLabel={t.common.delete}
        onConfirm={() => removeBank.mutate()}
        pending={removeBank.isPending}
        error={removeBank.error}
      />
      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(open) => !open && setDeleting(null)}
        title={t.banks.questionDeleteTitle}
        text={t.banks.questionDeleteText}
        confirmLabel={t.common.delete}
        onConfirm={() => deleting && removeQuestion.mutate(deleting)}
        pending={removeQuestion.isPending}
        error={removeQuestion.error}
      />
    </PageContainer>
  )
}

function BackLink() {
  const { t } = useI18n()
  const paths = usePaths()
  return (
    <Link
      to={paths.banks}
      className="inline-flex items-center gap-1.5 rounded-sm text-sm font-medium text-ink-muted hover:text-forest"
    >
      <ArrowLeft className="size-4" aria-hidden />
      {t.banks.title}
    </Link>
  )
}

function QuestionCard({
  number,
  question,
  onEdit,
  onDelete,
}: {
  number: number
  question: AuthorQuestion
  onEdit: () => void
  onDelete: () => void
}) {
  const { t } = useI18n()
  const hasCorrect = question.options.some((o) => o.is_correct)
  return (
    <li className="rounded-lg border border-line bg-surface p-4 sm:p-5">
      <div className="flex items-start gap-3 sm:gap-4">
        <span className="tabular mt-0.5 w-7 shrink-0 text-sm font-semibold text-ink-subtle">{number}.</span>
        <div className="min-w-0 flex-1">
          <QuestionText text={question.text} as="h3" />
          <ul className="mt-3 grid gap-1.5">
            {question.options.map((option, index) => (
              <li
                key={option.id}
                className={cn(
                  'flex items-start gap-2.5 rounded-md px-2.5 py-1.5 text-[0.9375rem]',
                  option.is_correct ? 'bg-forest-tint text-ink' : 'text-ink-muted',
                )}
              >
                <span
                  className={cn(
                    'grid size-6 shrink-0 place-items-center rounded-sm text-xs font-semibold',
                    option.is_correct ? 'bg-forest text-surface' : 'bg-sunken text-ink-muted',
                  )}
                  aria-hidden
                >
                  {option.is_correct ? <Check className="size-3.5" /> : optionLetter(index)}
                </span>
                <span className="min-w-0 flex-1 [overflow-wrap:anywhere]">
                  {option.is_correct && <span className="sr-only">{t.banks.correct}: </span>}
                  {option.text}
                </span>
                {option.is_correct && (
                  <span className="hidden shrink-0 text-xs font-medium text-forest sm:inline" aria-hidden>
                    {t.banks.correct}
                  </span>
                )}
              </li>
            ))}
          </ul>
          {!hasCorrect && <p className="mt-2 text-sm font-medium text-danger">{t.banks.noCorrect}</p>}
        </div>
        <div className="flex shrink-0 flex-col gap-1 sm:flex-row">
          <Button variant="ghost" size="icon-sm" onClick={onEdit} aria-label={`${t.banks.editQuestion} ${number}`}>
            <Pencil aria-hidden />
          </Button>
          <Button
            variant="danger-ghost"
            size="icon-sm"
            onClick={onDelete}
            aria-label={`${t.banks.deleteQuestion} ${number}`}
          >
            <Trash2 aria-hidden />
          </Button>
        </div>
      </div>
    </li>
  )
}
