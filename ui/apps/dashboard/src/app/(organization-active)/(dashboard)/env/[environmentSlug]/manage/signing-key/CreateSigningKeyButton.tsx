'use client';

import { Button } from '@inngest/components/Button';
import { RiAddLine } from '@remixicon/react';
import { useMutation } from 'urql';

import { graphql } from '@/gql';

const Mutation = graphql(`
  mutation CreateSigningKey($envID: UUID!) {
    createSigningKey(envID: $envID) {
      createdAt
    }
  }
`);

type Props = {
  disabled?: boolean;
  envID: string;
};

export function CreateSigningKeyButton({ disabled, envID }: Props) {
  const [, createSigningKey] = useMutation(Mutation);

  return (
    <Button
      btnAction={() => createSigningKey({ envID })}
      disabled={disabled}
      kind="primary"
      icon={<RiAddLine />}
      label="Create new signing key"
    />
  );
}
