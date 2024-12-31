'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';

import { EnvironmentType } from '@/gql/graphql';
import { EnvironmentArchiveModal } from './EnvironmentArchiveModal';

type Props = {
  env: { id: string; isArchived: boolean; name: string; type: EnvironmentType };
};

export function EnvironmentArchiveButton({ env }: Props) {
  const [isModalOpen, setIsModalOpen] = useState(false);

  // Need to store local state since the mutations don't invalidate the cache,
  // which means that the env prop won't update. We should fix cache
  // invalidation
  const [isArchived, setIsArchived] = useState(env.isArchived);

  let label;
  if (isArchived) {
    label = 'Unarchive';
  } else {
    label = 'Archive';
  }

  return (
    <>
      <Button
        appearance="outlined"
        onClick={() => setIsModalOpen(true)}
        kind="danger"
        label={label}
      />

      <EnvironmentArchiveModal
        envID={env.id}
        isArchived={isArchived}
        isBranchEnv={env.type === EnvironmentType.BranchChild}
        isOpen={isModalOpen}
        onCancel={() => setIsModalOpen(false)}
        onSuccess={() => {
          setIsModalOpen(false);
          setIsArchived(!isArchived);
        }}
      />
    </>
  );
}
