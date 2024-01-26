'use client';

import React, { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { IconReplay } from '@inngest/components/icons/Replay';

import NewReplayModal from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/NewReplayModal';

type NewReplayButtonProps = {
  functionSlug: string;
};

export default function NewReplayButton({ functionSlug }: NewReplayButtonProps) {
  const [isModalVisible, setIsModalVisible] = useState(false);

  return (
    <>
      <Button
        label="New Replay"
        kind="primary"
        btnAction={() => setIsModalVisible(true)}
        icon={<IconReplay className="h-5 w-5 text-white" />}
      />
      <NewReplayModal
        isOpen={isModalVisible}
        functionSlug={functionSlug}
        onClose={() => setIsModalVisible(false)}
      />
    </>
  );
}
