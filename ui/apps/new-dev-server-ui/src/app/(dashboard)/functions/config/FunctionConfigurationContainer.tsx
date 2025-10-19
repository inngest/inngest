'use client'

import { useRouter } from 'next/navigation'
import { Button } from '@inngest/components/Button'
import { FunctionConfiguration } from '@inngest/components/FunctionConfiguration'
import { Header } from '@inngest/components/Header/Header'
import { InvokeButton } from '@inngest/components/InvokeButton'
import { RiCloseLine } from '@remixicon/react'
import { toast } from 'sonner'

import {
  FunctionTriggerTypes,
  useGetFunctionQuery,
  useInvokeFunctionMutation,
} from '@/store/generated'

type FunctionDetailsProps = {
  onClose: () => void
  functionSlug: string
}

export function FunctionConfigurationContainer({
  onClose,
  functionSlug,
}: FunctionDetailsProps) {
  const { data, isFetching } = useGetFunctionQuery(
    { functionSlug: functionSlug },
    {
      refetchOnMountOrArgChange: true,
    },
  )

  const [invokeFunction] = useInvokeFunctionMutation()

  const triggers = data?.functionBySlug?.triggers

  const router = useRouter()
  const doesFunctionAcceptPayload = Boolean(
    triggers?.some((trigger) => trigger.type === FunctionTriggerTypes.Event),
  )

  if (isFetching || !data || !data.functionBySlug) {
    return null
  }

  // TypeScript flow analysis helper - function is guaranteed to exist after the null check above
  const inngestFunction = data.functionBySlug

  const header = (
    <Header
      breadcrumb={[{ text: inngestFunction.name }]}
      action={
        <div className="flex flex-row items-center justify-end gap-2">
          <InvokeButton
            kind="primary"
            appearance="solid"
            disabled={false}
            doesFunctionAcceptPayload={doesFunctionAcceptPayload}
            btnAction={async ({ data, user }) => {
              await invokeFunction({
                data,
                functionSlug: inngestFunction.slug,
                user,
              })
              toast.success('Function invoked')
              router.push('/runs')
            }}
          />
          <Button
            icon={<RiCloseLine className="text-muted h-5 w-5" />}
            kind="secondary"
            appearance="ghost"
            size="small"
            onClick={() => onClose()}
          />
        </div>
      }
    />
  )

  return (
    <FunctionConfiguration
      inngestFunction={data.functionBySlug}
      header={header}
      getFunctionLink={(functionSlug) =>
        `/functions/config?slug=${functionSlug}`
      }
    />
  )
}
