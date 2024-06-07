'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { RiArchive2Line } from '@remixicon/react';

import ArchiveEventModal from './ArchiveEventModal';

type ArchiveButtonProps = {
  eventName: string;
};

export default function ArchiveEventButton({ eventName }: ArchiveButtonProps) {
  const [isArchiveEventModalVisible, setIsArchiveEventModalVisible] = useState<boolean>(false);
  return (
    <>
      <Button
        icon={<RiArchive2Line />}
        appearance="outlined"
        onClick={() => setIsArchiveEventModalVisible(true)}
        label="Archive Event"
      />
      <ArchiveEventModal
        eventName={eventName}
        isOpen={isArchiveEventModalVisible}
        onClose={() => setIsArchiveEventModalVisible(false)}
      />
    </>
  );
}
