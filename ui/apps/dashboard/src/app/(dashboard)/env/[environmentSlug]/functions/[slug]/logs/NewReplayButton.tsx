'use client';

import React, { useState } from 'react';
import { Button } from '@inngest/components/Button';

import NewReplayModal from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/NewReplayModal';
import ReplayIcon from '@/icons/replay.svg';

type NewReplayButtonProps = {
  environmentSlug: string;
  functionSlug?: string;
};

export default function NewReplayButton({ environmentSlug, functionSlug }: NewReplayButtonProps) {
  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <>
      <Button
        label="New Replay"
        kind="primary"
        btnAction={() => setIsModalVisible(true)}
        icon={<ReplayIcon className="h-5 w-5 text-white" />}
      />
      <NewReplayModal
        isOpen={isModalVisible}
        environmentSlug={environmentSlug}
        functionSlug={functionSlug}
        onClose={() => setIsModalVisible(false)}
      />
    </>
  );
}
