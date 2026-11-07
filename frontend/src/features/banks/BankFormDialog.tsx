import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { AlertCircle } from 'lucide-react'
import type { z } from 'zod'
import { useApi } from '@/api/context'
import { isApiError } from '@/api/errors'
import { queryKeys } from '@/api/queryKeys'
import type { QuestionBank } from '@/api/types'
import { useT } from '@/i18n'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Field } from '@/components/common/forms'
import { errorMessage } from '@/components/common/states'
import { useToast } from '@/components/common/toast'
import { bankSchema } from './schemas'

type BankValues = z.infer<ReturnType<typeof bankSchema>>

/** Create (no `bank`) or edit a bank. Creation is never retried on its own. */
export function BankFormDialog({
  open,
  onOpenChange,
  bank,
  onCreated,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  bank?: QuestionBank
  onCreated?: (bank: QuestionBank) => void
}) {
  const t = useT()
  const api = useApi()
  const toast = useToast()
  const queryClient = useQueryClient()
  const form = useForm<BankValues>({
    resolver: zodResolver(bankSchema(t)),
    values: { title: bank?.title ?? '', description: bank?.description ?? '' },
  })

  const save = useMutation({
    mutationFn: (values: BankValues) =>
      bank ? api.banks.update(bank.id, values) : api.banks.create(values),
    onSuccess: (saved) => {
      queryClient.setQueryData(queryKeys.bank(saved.id), saved)
      void queryClient.invalidateQueries({ queryKey: queryKeys.banks, exact: true })
      toast.success(bank ? t.banks.saved : t.banks.created)
      close()
      if (!bank) onCreated?.(saved)
    },
  })

  const errors = form.formState.errors
  function close() {
    form.reset()
    save.reset()
    onOpenChange(false)
  }
  const uncertain = !bank && isApiError(save.error, 'network')

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (save.isPending) return
        if (next) onOpenChange(true)
        else close()
      }}
    >
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{bank ? t.banks.editTitle : t.banks.createTitle}</DialogTitle>
          {!bank && <DialogDescription>{t.banks.createText}</DialogDescription>}
        </DialogHeader>
        <form
          id="bank-form"
          noValidate
          className="flex flex-col gap-5"
          onSubmit={form.handleSubmit((values) => {
            if (!save.isPending) save.mutate(values)
          })}
        >
          {save.isError && (
            <Alert tone={uncertain ? 'warning' : 'danger'}>
              <AlertCircle aria-hidden />
              <AlertDescription>{uncertain ? t.banks.createUnknown : errorMessage(t, save.error)}</AlertDescription>
            </Alert>
          )}
          <Field label={t.banks.fieldTitle} error={errors.title?.message}>
            {(control) => (
              <Input {...control} placeholder={t.banks.fieldTitlePlaceholder} {...form.register('title')} />
            )}
          </Field>
          <Field label={t.banks.fieldDescription} optional error={errors.description?.message}>
            {(control) => (
              <Textarea
                {...control}
                rows={3}
                placeholder={t.banks.fieldDescriptionPlaceholder}
                {...form.register('description')}
              />
            )}
          </Field>
        </form>
        <DialogFooter>
          <Button variant="secondary" onClick={close} disabled={save.isPending}>
            {t.common.cancel}
          </Button>
          <Button type="submit" form="bank-form" loading={save.isPending}>
            {bank ? t.common.saveChanges : t.common.create}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
