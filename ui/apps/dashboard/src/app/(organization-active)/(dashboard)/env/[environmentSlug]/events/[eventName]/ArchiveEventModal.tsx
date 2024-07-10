'use client';

import { Button } from '@inngest/components/Button';
import { useMutation } from 'urql';

import { useEnvironment } from '@/components/Environments/EnvContext';
import Modal from '@/components/Modal';
import { graphql } from '@/gql';

const ArchiveEvent = graphql(`
  mutation ArchiveEvent($environmentId: ID!, $name: String!) {
    archiveEvent(workspaceID: $environmentId, name: $name) {
      name
    }
  }
`);

type ArchiveEventModalProps = {
  eventName: string;
  isOpen: boolean;
  onClose: () => void;
};

export default function ArchiveEventModal({ eventName, isOpen, onClose }: ArchiveEventModalProps) {
  const environment = useEnvironment();
  const missingData = !eventName;
  const [, archiveEvent] = useMutation(ArchiveEvent);

  return (
    <Modal className="flex max-w-xl flex-col gap-4" isOpen={isOpen} onClose={onClose}>
      <p className="pb-4">Are you sure you want to archive this event?</p>
      <div className="flex content-center justify-end">
        <Button appearance="outlined" btnAction={() => onClose()} label="No" />
        <Button
          kind="danger"
          appearance="text"
          disabled={missingData}
          btnAction={() => {
            !missingData && archiveEvent({ name: eventName, environmentId: environment.id });
            !missingData && onClose();
          }}
          label="Yes"
        />
      </div>
    </Modal>
  );
}
