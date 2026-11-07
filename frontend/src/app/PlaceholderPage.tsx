import { Link } from 'react-router'
import { usePaths } from '@/app/paths'
import { useT } from '@/i18n'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/common/states'

/** Sections that arrive in phase 2. */
export function PlaceholderPage() {
  const t = useT()
  const paths = usePaths()
  return (
    <div className="mx-auto max-w-content px-4 pt-10 md:px-8 lg:px-10">
      <EmptyState
        title="Раздел появится на втором этапе"
        text="Сейчас для оценки дизайна готовы кабинет и экран прохождения теста."
        action={
          <Button asChild variant="secondary">
            <Link to={paths.dashboard}>{t.nav.dashboard}</Link>
          </Button>
        }
      />
    </div>
  )
}
