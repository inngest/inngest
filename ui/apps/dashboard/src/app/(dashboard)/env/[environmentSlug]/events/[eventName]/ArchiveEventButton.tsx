'use client';

import { useState } from 'react';
import { ArchiveBoxIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';

import ArchiveEventModal from './ArchiveEventModal';

type ArchiveButtonProps = {
  environmentSlug: string;
  eventName: string;
};

export default function ArchiveEventButton({ environmentSlug, eventName }: ArchiveButtonProps) {
  const [isArchiveEventModalVisible, setIsArchiveEventModalVisible] = useState<boolean>(false);
  return (
    <>
      <Button
        icon={<ArchiveBoxIcon />}
        appearance="outlined"
        btnAction={() => setIsArchiveEventModalVisible(true)}
        label="Archive Event"
      />
      <ArchiveEventModal
        environmentSlug={environmentSlug}
        eventName={eventName}
        isOpen={isArchiveEventModalVisible}
        onClose={() => setIsArchiveEventModalVisible(false)}
      />
    </>
  );
}
