'use client';

import { AlertModal } from '@inngest/components/Modal/AlertModal';
import { useMutation } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
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
    <AlertModal
      className="w-1/3 max-w-xl"
      isOpen={isOpen}
      title="Are you sure you want to archive this event?"
      onClose={onClose}
      onSubmit={() => {
        !missingData && archiveEvent({ name: eventName, environmentId: environment.id });
        !missingData && onClose();
      }}
    />
  );
}
