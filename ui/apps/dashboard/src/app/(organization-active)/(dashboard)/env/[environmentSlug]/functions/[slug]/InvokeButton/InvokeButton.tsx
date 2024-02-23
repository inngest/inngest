'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { IconTriggerFunction } from '@inngest/components/icons/TriggerFunction';
import { useMutation } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { graphql } from '@/gql';
import { InvokeModal } from './InvokeModal';

const InvokeFunctionDocument = graphql(`
  mutation InvokeFunction($envID: UUID!, $data: Map, $functionSlug: String!) {
    invokeFunction(envID: $envID, data: $data, functionSlug: $functionSlug)
  }
`);

type Props = {
  disabled: boolean;
  functionSlug: string;
  doesFunctionAcceptPayload: boolean;
};

export function InvokeButton({
  disabled,
  functionSlug,
  doesFunctionAcceptPayload: hasEventTrigger,
}: Props) {
  const env = useEnvironment();
  const [, invokeFunction] = useMutation(InvokeFunctionDocument);
  const [isModalOpen, setIsModalOpen] = useState(false);

  function onConfirm({ data }: { data: Record<string, unknown> }) {
    setIsModalOpen(false);
    invokeFunction({
      envID: env.id,
      data,
      functionSlug,
    });
  }

  return (
    <>
      <Button
        btnAction={() => setIsModalOpen(true)}
        disabled={disabled}
        icon={<IconTriggerFunction />}
        label="Invoke"
      />

      <InvokeModal
        doesFunctionAcceptPayload={hasEventTrigger}
        isOpen={isModalOpen}
        onCancel={() => setIsModalOpen(false)}
        onConfirm={onConfirm}
      />
    </>
  );
}
