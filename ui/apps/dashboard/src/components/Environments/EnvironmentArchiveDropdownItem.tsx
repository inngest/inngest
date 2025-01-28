'use client';

import { useState } from 'react';
import { DropdownMenuItem } from '@inngest/components/DropdownMenu';
import { RiArchive2Line } from '@remixicon/react';

import { EnvironmentType } from '@/gql/graphql';
import { EnvironmentArchiveModal } from './EnvironmentArchiveModal';

type Props = {
  env: { id: string; isArchived: boolean; name: string; type: EnvironmentType };
  onClose: () => void;
};

export function EnvironmentArchiveDropdownItem({ env, onClose }: Props) {
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
      <DropdownMenuItem
        onSelect={(e) => {
          e.preventDefault();
          setIsModalOpen(true);
        }}
        className={!isArchived ? 'text-error' : undefined}
      >
        <RiArchive2Line className="h-4 w-4" />
        {label}
      </DropdownMenuItem>

      <EnvironmentArchiveModal
        envID={env.id}
        isArchived={isArchived}
        isBranchEnv={env.type === EnvironmentType.BranchChild}
        isOpen={isModalOpen}
        onCancel={() => {
          setIsModalOpen(false);
          onClose();
        }}
        onSuccess={() => {
          setIsModalOpen(false);
          setIsArchived(!isArchived);
          onClose();
        }}
      />
    </>
  );
}
