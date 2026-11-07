import { useMemo } from 'react'
import { Link } from 'react-router'
import { Plus } from 'lucide-react'
import { usePaths } from '@/app/paths'
import { useI18n } from '@/i18n'
import { Button } from '@/components/ui/button'
import { EmptyState, ErrorState, ListSkeleton, PageContainer, PageHeader, Panel } from '@/components/common/states'
import { useBanks } from '@/features/banks/hooks'
import { QuizRowWithHistory } from './QuizRow'
import { useQuizList } from './hooks'

export function QuizzesPage() {
  const { t } = useI18n()
  const paths = usePaths()
  const quizzes = useQuizList()
  const banks = useBanks()
  const bankById = useMemo(() => new Map((banks.data ?? []).map((bank) => [bank.id, bank])), [banks.data])
  const items = quizzes.data?.pages.flatMap((page) => page.items) ?? []

  return (
    <PageContainer>
      <PageHeader
        title={t.quizzes.title}
        subtitle={t.quizzes.subtitle}
        actions={
          <Button asChild>
            <Link to={paths.newQuiz}>
              <Plus aria-hidden />
              {t.quizzes.create}
            </Link>
          </Button>
        }
      />
      <Panel className="mt-8">
        {quizzes.isPending ? (
          <ListSkeleton rows={4} />
        ) : quizzes.isError ? (
          <ErrorState
            title={t.quizzes.loadError}
            error={quizzes.error}
            onRetry={() => void quizzes.refetch()}
            retrying={quizzes.isFetching}
          />
        ) : items.length === 0 ? (
          <EmptyState
            title={t.quizzes.emptyTitle}
            text={t.quizzes.emptyText}
            action={
              <Button asChild>
                <Link to={paths.newQuiz}>
                  <Plus aria-hidden />
                  {t.quizzes.create}
                </Link>
              </Button>
            }
          />
        ) : (
          <ul className="divide-y divide-line-soft">
            {items.map((quiz) => (
              <QuizRowWithHistory key={quiz.id} quiz={quiz} bank={bankById.get(quiz.question_bank_id)} />
            ))}
          </ul>
        )}
      </Panel>
      {quizzes.hasNextPage && (
        <div className="mt-4 flex justify-center">
          <Button
            variant="secondary"
            onClick={() => void quizzes.fetchNextPage()}
            loading={quizzes.isFetchingNextPage}
          >
            {t.common.showMore}
          </Button>
        </div>
      )}
    </PageContainer>
  )
}
