import { useState } from 'react';
import { RiCheckLine, RiCheckboxCircleFill, RiExternalLinkLine } from '@remixicon/react';
import { toast } from 'sonner';

import { Alert } from '../Alert';
import { Link } from '../Link';
import { AlertModal } from '../Modal';
import { useRerun } from '../Shared/useRerun';

type RerunProps = {
  runID: string;
  fnID?: string;
  open: boolean;
  onClose: () => void;
};
export const RerunModal = ({ runID, fnID, open, onClose }: RerunProps) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const { rerun } = useRerun();

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
        const { data, error, redirect } = await rerun({ runID, fnID });
        setError(error ?? null);

        if (data?.newRunID) {
          toast.success(
            <Link
              size="medium"
              href={redirect ?? ''}
              iconBefore={
                <RiCheckboxCircleFill className="bg-success dark:bg-success/40 text-success h-4 w-4 shrink-0" />
              }
              iconAfter={<RiExternalLinkLine className="h-4 w-4 shrink-0" />}
              className="z-50 flex flex-row items-center gap-2"
            >
              Successfully queued rerun
            </Link>
          );

          close();
        }
        setLoading(false);
      }}
      title={`Are you sure you want to rerun this function?`}
      className="w-[600px]"
      confirmButtonLabel="Rerun"
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
