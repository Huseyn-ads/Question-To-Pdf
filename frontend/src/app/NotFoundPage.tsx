import { Link } from 'react-router'
import { usePaths } from '@/app/paths'
import { useT } from '@/i18n'
import { Button } from '@/components/ui/button'
import { EmptyState, PageContainer, Panel } from '@/components/common/states'

export function NotFoundPage() {
  const t = useT()
  const paths = usePaths()
  return (
    <PageContainer>
      <Panel>
        <EmptyState
          title={t.notFound.title}
          text={t.notFound.text}
          action={
            <Button asChild>
              <Link to={paths.dashboard}>{t.common.toDashboard}</Link>
            </Button>
          }
        />
      </Panel>
    </PageContainer>
  )
}
