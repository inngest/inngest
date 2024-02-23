import { useEffect, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';

type Props = {
  isArchived: boolean;
  isOpen: boolean;
  onCancel: () => void;
  onConfirm: () => void;
};

export function EnvironmentArchiveModal(props: Props) {
  const { isOpen, onCancel, onConfirm } = props;

  // Use an internal isArchived to prevent text changes after
  // confirmation.
  const [isArchived, setIsArchived] = useState(props.isArchived);
  useEffect(() => {
    if (isOpen) {
      setIsArchived(props.isArchived);
    }
  }, [isOpen, props.isArchived]);

  let title: string;
  if (isArchived) {
    title = 'Unarchive environment';
  } else {
    title = 'Archive environment';
  }

  let content: string;
  if (isArchived) {
    content = 'Any active functions within this environment will become triggerable.';
  } else {
    content =
      'Functions within this environment will no longer be triggerable. Nothing will be deleted and you can unarchive at any time.';
  }

  return (
    <Modal className="flex max-w-xl flex-col gap-4" isOpen={isOpen} onClose={onCancel}>
      <Modal.Header>{title}</Modal.Header>

      <Modal.Body className="text-sm">{content}</Modal.Body>

      <Modal.Footer className="flex content-center justify-end">
        <Button appearance="outlined" btnAction={onCancel} label="Cancel" />

        <Button
          kind="danger"
          appearance="text"
          btnAction={onConfirm}
          label={isArchived ? 'Unarchive' : 'Archive'}
        />
      </Modal.Footer>
    </Modal>
  );
}
