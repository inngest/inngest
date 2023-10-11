import { useEffect, useState } from 'react';

import Button from '@/components/Button';
import Modal from '@/components/Modal';

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

  return (
    <Modal className="flex max-w-xl flex-col gap-4" isOpen={isOpen} onClose={onCancel}>
      <p>{`Are you sure you want to ${isArchived ? 'unarchive' : 'archive'} this environment?`}</p>

      {isArchived && (
        <p className="pb-4 text-sm">
          Any active functions within this environment will become triggerable.
        </p>
      )}

      {!isArchived && (
        <p className="pb-4 text-sm">
          Functions within this environment will no longer be triggerable. Nothing will be deleted
          and you can unarchive at any time.
        </p>
      )}

      <div className="flex content-center justify-end">
        <Button variant="secondary" onClick={onCancel}>
          Cancel
        </Button>

        <Button variant="text-danger" onClick={onConfirm}>
          {isArchived ? 'Unarchive' : 'Archive'}
        </Button>
      </div>
    </Modal>
  );
}
