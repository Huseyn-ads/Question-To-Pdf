import { useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { BookOpenText, ChevronRight, FileUp, Plus } from 'lucide-react'
import { usePaths } from '@/app/paths'
import { useI18n } from '@/i18n'
import { Button } from '@/components/ui/button'
import { EmptyState, ErrorState, ListSkeleton, PageContainer, PageHeader, Panel } from '@/components/common/states'
import { BankFormDialog } from './BankFormDialog'
import { useBanks } from './hooks'

export function BanksPage() {
  const { t, formatRelativeDay } = useI18n()
  const paths = usePaths()
  const navigate = useNavigate()
  const banks = useBanks()
  const [createOpen, setCreateOpen] = useState(false)

  return (
    <PageContainer>
      <PageHeader
        title={t.banks.title}
        subtitle={t.banks.subtitle}
        actions={
          <>
            <Button variant="secondary" asChild>
              <Link to={paths.importDocx}>
                <FileUp aria-hidden />
                {t.banks.importDocx}
              </Link>
            </Button>
            <Button onClick={() => setCreateOpen(true)}>
              <Plus aria-hidden />
              {t.banks.create}
            </Button>
          </>
        }
      />

      <Panel className="mt-8">
        {banks.isPending ? (
          <ListSkeleton rows={4} compact />
        ) : banks.isError ? (
          <ErrorState
            title={t.banks.loadError}
            error={banks.error}
            onRetry={() => void banks.refetch()}
            retrying={banks.isFetching}
          />
        ) : banks.data.length === 0 ? (
          <EmptyState
            title={t.banks.emptyTitle}
            text={t.banks.emptyText}
            action={
              <>
                <Button asChild>
                  <Link to={paths.importDocx}>
                    <FileUp aria-hidden />
                    {t.banks.importDocx}
                  </Link>
                </Button>
                <Button variant="secondary" onClick={() => setCreateOpen(true)}>
                  <Plus aria-hidden />
                  {t.banks.create}
                </Button>
              </>
            }
          />
        ) : (
          <ul className="divide-y divide-line-soft">
            {banks.data.map((bank) => (
              <li key={bank.id}>
                <Link
                  to={paths.bank(bank.id)}
                  className="group flex items-center gap-4 px-4 py-4 hover:bg-forest-tint/60 sm:px-5"
                >
                  <span className="grid size-10 shrink-0 place-items-center rounded-md bg-sage-soft text-sage-ink">
                    <BookOpenText className="size-5" aria-hidden />
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="line-clamp-2 font-medium text-ink [overflow-wrap:anywhere] group-hover:text-forest">
                      {bank.title}
                    </span>
                    {bank.description && (
                      <span className="mt-0.5 line-clamp-2 text-sm text-ink-muted [overflow-wrap:anywhere]">
                        {bank.description}
                      </span>
                    )}
                    <span className="mt-0.5 block text-sm text-ink-subtle">
                      {t.banks.updated(formatRelativeDay(bank.updated_at))}
                    </span>
                  </span>
                  <ChevronRight className="size-5 shrink-0 text-ink-subtle" aria-hidden />
                </Link>
              </li>
            ))}
          </ul>
        )}
      </Panel>

      <BankFormDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreated={(bank) => void navigate(paths.bank(bank.id))}
      />
    </PageContainer>
  )
}
