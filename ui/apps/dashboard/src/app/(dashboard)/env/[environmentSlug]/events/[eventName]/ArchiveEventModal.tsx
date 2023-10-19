'use client';

import { Button } from '@inngest/components/Button';
import { useMutation } from 'urql';

import Modal from '@/components/Modal';
import { graphql } from '@/gql';
import { useEnvironment } from '@/queries';

const ArchiveEvent = graphql(`
  mutation ArchiveEvent($environmentId: ID!, $name: String!) {
    archiveEvent(workspaceID: $environmentId, name: $name) {
      name
    }
  }
`);

type ArchiveEventModalProps = {
  environmentSlug: string;
  eventName: string;
  isOpen: boolean;
  onClose: () => void;
};

export default function ArchiveEventModal({
  environmentSlug,
  eventName,
  isOpen,
  onClose,
}: ArchiveEventModalProps) {
  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug,
  });
  const environmentId = environment?.id;
  const missingData = isFetchingEnvironment || !eventName || !environmentId;
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
            !missingData && archiveEvent({ name: eventName, environmentId });
            !missingData && onClose();
          }}
          label="Yes"
        />
      </div>
    </Modal>
  );
}
