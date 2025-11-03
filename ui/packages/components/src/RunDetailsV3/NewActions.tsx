import { useState } from 'react';
import { RiCheckboxCircleFill, RiExternalLinkLine } from '@remixicon/react';
import { toast } from 'sonner';

import { Button } from '../Button/NewButton';
import { Link } from '../Link/NewLink';
import { useShared } from '../SharedContext/SharedContext';
import { useCancelRun } from '../SharedContext/useCancelRun';
import { useRerun } from '../SharedContext/useRerun';
import { OptionalTooltip } from '../Tooltip/OptionalTooltip';
import { CancelModal } from './CancelModal';

export type RunActions = {
  allowCancel?: boolean;
  runID: string;
  fnID?: string;
};

export const Actions = ({ allowCancel, runID, fnID }: RunActions) => {
  const [rerunLoading, setRerunLoading] = useState(false);
  const [cancelLoading, setCancelLoading] = useState(false);
  const [cancelOpen, setCancelOpen] = useState(false);
  const { rerun } = useRerun();
  const { cancelRun } = useCancelRun();
  const { cloud } = useShared();

  return (
    <div className="flex flex-row items-center justify-end gap-2">
      <CancelModal runID={runID} open={cancelOpen} onClose={() => setCancelOpen(false)} />
      <Button
        kind="primary"
        appearance="outlined"
        size="medium"
        label="Rerun"
        loading={rerunLoading}
        disabled={rerunLoading}
        onClick={async () => {
          setRerunLoading(true);
          const { data, error, redirect } = await rerun({ runID, fnID });

          if (error) {
            toast.error('Rerun failed. Please try again.');
          }

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
          }
          setRerunLoading(false);
        }}
      />
      <OptionalTooltip tooltip={!allowCancel && 'Only active runs can be cancelled'}>
        <Button
          kind="danger"
          appearance="outlined"
          size="medium"
          iconSide="left"
          label="Cancel"
          loading={cancelLoading}
          disabled={!allowCancel || cancelLoading}
          onClick={async () => {
            if (cloud) {
              setCancelOpen(true);
              return;
            }

            setCancelLoading(true);

            const { data, error } = await cancelRun({ runID });
            if (error) {
              toast.error('Failed to cancel run');
            }
            if (data?.cancelRun?.id) {
              toast.success('Run cancelled!');
            }

            setCancelLoading(false);
          }}
        />
      </OptionalTooltip>
    </div>
  );
};
