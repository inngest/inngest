'use client';

import React, { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { IconReplay } from '@inngest/components/icons/Replay';

import NewReplayModal from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/NewReplayModal';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';

type NewReplayButtonProps = {
  functionSlug: string;
};

export default function NewReplayButton({ functionSlug }: NewReplayButtonProps) {
  const [isModalVisible, setIsModalVisible] = useState(false);
  const { value: newIANav, isReady } = useBooleanFlag('new-ia-nav');

  return (
    <>
      {isReady && !newIANav && (
        <Button
          label="New Replay"
          kind="primary"
          btnAction={() => setIsModalVisible(true)}
          icon={<IconReplay className="h-5 w-5 py-2 text-white" />}
          className="my-2"
        />
      )}
      <NewReplayModal
        isOpen={isModalVisible}
        functionSlug={functionSlug}
        onClose={() => setIsModalVisible(false)}
      />
    </>
  );
}
