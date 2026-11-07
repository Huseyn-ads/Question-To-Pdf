import { AlertCircle } from 'lucide-react'
import { useT } from '@/i18n'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { errorMessage } from './states'

/** Destructive confirmation. Stays open on failure and shows why. */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  text,
  confirmLabel,
  onConfirm,
  pending,
  error,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  text: string
  confirmLabel: string
  onConfirm: () => void
  pending: boolean
  error: unknown
}) {
  const t = useT()
  return (
    <Dialog open={open} onOpenChange={(next) => !pending && onOpenChange(next)}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription className="[overflow-wrap:anywhere]">{text}</DialogDescription>
        </DialogHeader>
        {error != null && (
          <Alert tone="danger">
            <AlertCircle aria-hidden />
            <AlertDescription>{errorMessage(t, error)}</AlertDescription>
          </Alert>
        )}
        <DialogFooter>
          <Button variant="secondary" onClick={() => onOpenChange(false)} disabled={pending}>
            {t.common.cancel}
          </Button>
          <Button variant="danger" onClick={onConfirm} loading={pending}>
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
