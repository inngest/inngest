import { useCallback, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { NewButton } from '@inngest/components/Button';
import { RangePicker } from '@inngest/components/DatePicker';
import { Modal } from '@inngest/components/Modal';
import { subtractDuration } from '@inngest/components/utils/date';

import Input from '@/components/Forms/Input';
import { Label } from '@/components/Forms/Label';
import { usePlanFeatures } from '../runs/usePlanFeatures';
import { useCreateCancellation } from './useCreateCancellation';
import { useRunCount, type RunCountInput } from './useRunCount';

type Props = {
  envID: string;
  functionSlug: string;
  isOpen: boolean;
  onClose: () => void;
};

export function CancelFunctionModal(props: Props) {
  const { envID, functionSlug, isOpen, onClose: _onClose } = props;
  const [creationError, setCreationError] = useState<Error>();
  const [isCreating, setIsCreating] = useState<boolean>(false);
  const [name, setName] = useState<string>();
  const [timeRange, setTimeRange] = useState<{ start: Date; end: Date }>();
  const createCancellation = useCreateCancellation({ functionSlug });

  const onClose = useCallback(() => {
    setCreationError(undefined);
    setIsCreating(false);
    setName(undefined);
    setTimeRange(undefined);

    _onClose();
  }, [_onClose]);

  const onSubmit = useCallback(async () => {
    if (!timeRange) {
      return;
    }

    try {
      const res = await createCancellation({
        name,
        queuedAtMax: timeRange.end,
        queuedAtMin: timeRange.start,
      });
      if (res.error) {
        throw res.error;
      }
      onClose();
    } catch (e) {
      if (e instanceof Error) {
        setCreationError(e);
      } else {
        setCreationError(new Error('unknown error'));
      }
      return;
    } finally {
      setIsCreating(false);
    }
  }, [createCancellation, name, onClose, timeRange]);

  let runCountInput: RunCountInput | undefined;
  if (timeRange) {
    runCountInput = {
      envID,
      functionSlug,
      queuedAtMin: timeRange.start,
      queuedAtMax: timeRange.end,
    };
  }

  const countRes = useRunCount(runCountInput);
  const featureRes = usePlanFeatures();

  let upgradeCutoff: Date | undefined;
  if (featureRes.data) {
    upgradeCutoff = subtractDuration(new Date(), { days: featureRes.data.history });
  }

  return (
    <Modal className="w-[800px]" isOpen={isOpen} onClose={onClose}>
      <Modal.Header>Bulk cancellation</Modal.Header>

      <Modal.Body>
        <div className="flex flex-col gap-4">
          <div>
            <Label name="cancellation-name" optional>
              Name
            </Label>

            <Input
              name="cancellation-name"
              placeholder="My cancellation"
              value={name}
              onChange={(e) => {
                setName(e.target.value);
              }}
            />
          </div>

          <div>
            <Label
              description="Choose the time range when the function runs were queued"
              name="time-range"
            >
              Date range
            </Label>

            <RangePicker
              className="w-full"
              disabled={featureRes.isLoading}
              onChange={(range) =>
                setTimeRange(
                  range.type === 'relative'
                    ? { start: subtractDuration(new Date(), range.duration), end: new Date() }
                    : { start: range.start, end: range.end }
                )
              }
              upgradeCutoff={upgradeCutoff}
            />
          </div>

          <Alert severity="info">
            This action will affect only queued and running function runs. All affected function
            runs will immediately cancel, but their status may not update immediately.
          </Alert>

          {countRes.error && (
            <Alert severity="error">Failed to query run count: {countRes.error.message}</Alert>
          )}

          {featureRes.error && (
            <Alert severity="error">
              Failed to query plan features: {featureRes.error.message}
            </Alert>
          )}

          {creationError && (
            <Alert severity="error">Failed to create cancellation: {creationError.message}</Alert>
          )}
        </div>
      </Modal.Body>

      <Modal.Footer className="flex gap-2">
        <div className="grow">
          <FooterMessage count={countRes.data} />
        </div>

        <NewButton
          appearance="outlined"
          disabled={isCreating}
          kind="secondary"
          label="Close"
          onClick={onClose}
        />
        <NewButton
          disabled={isCreating || !timeRange || countRes.data === 0}
          kind="danger"
          label="Submit"
          onClick={onSubmit}
        />
      </Modal.Footer>
    </Modal>
  );
}

function FooterMessage({ count }: { count: number | undefined }) {
  if (count === undefined) {
    return null;
  }

  if (count === 0) {
    return <span>No runs to cancel</span>;
  }

  if (count === 1) {
    return <span>Approximately 1 run will cancell</span>;
  }

  return <span>Approximately {count} runs will cancel</span>;
}
