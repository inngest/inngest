import { useState } from 'react';
import { toast } from 'sonner';

import { Alert } from '../Alert';
import { AlertModal } from '../Modal';
import { useCancelRun } from '../SharedContext/useCancelRun';

type RerunProps = {
  runID: string;
  open: boolean;
  onClose: () => void;
};
export const CancelModal = ({ runID, open, onClose }: RerunProps) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const { cancelRun } = useCancelRun();

  const close = () => {
    setLoading(false);
    setError(null);
    onClose();
  };

  return (
    <AlertModal
      isLoading={loading}
      isOpen={open}
      onClose={close}
      autoClose={false}
      onSubmit={async () => {
        setLoading(true);
        const { data, error } = await cancelRun({ runID });
        setError(error ?? null);

        if (data?.cancelRun?.id) {
          toast.success('Run cancelled!');
          close();
        }

        setLoading(false);
      }}
      title={`Are you sure you want to cancel this function?`}
      className="w-[600px]"
      confirmButtonKind="primary"
    >
      {error && (
        <Alert className="mt-4" severity="error">
          {error.message}
        </Alert>
      )}
    </AlertModal>
  );
};
