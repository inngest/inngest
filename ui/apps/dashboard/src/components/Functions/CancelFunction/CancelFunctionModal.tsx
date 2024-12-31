import { useCallback, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { RangePicker } from '@inngest/components/DatePicker';
import { Input } from '@inngest/components/Forms/Input';
import { Modal } from '@inngest/components/Modal';
import { subtractDuration } from '@inngest/components/utils/date';

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

  return (
    <Modal className="w-[800px]" isOpen={isOpen} onClose={onClose}>
      <Modal.Header>Bulk cancellation</Modal.Header>

      <Modal.Body>
        <div className="flex flex-col gap-4">
          <div>
            <label className="text-basis mb-1 text-sm font-medium" htmlFor="cancellation-name">
              Name (optional)
            </label>
            <p className="text-muted mb-1 text-sm">Provide a name for this cancellation group</p>

            <Input
              name="cancellation-name"
              onChange={(e) => {
                setName(e.target.value);
              }}
              optional
              placeholder="My cancellation"
              value={name}
            />
          </div>

          <div>
            <label className="text-basis mb-1 text-sm font-medium" htmlFor="time-range">
              Date range
            </label>
            <p className="text-muted mb-1 text-sm">
              Choose the time range when the function runs were queued
            </p>

            <RangePicker
              className="w-full"
              onChange={(range) =>
                setTimeRange(
                  range.type === 'relative'
                    ? { start: subtractDuration(new Date(), range.duration), end: new Date() }
                    : { start: range.start, end: range.end }
                )
              }
            />
          </div>

          <Alert severity="info" className="text-sm">
            This action will affect only queued and running function runs. All affected function
            runs will immediately cancel, but their status may not update immediately.
          </Alert>

          {countRes.error && (
            <Alert severity="error" className="text-sm">
              Failed to query run count: {countRes.error.message}
            </Alert>
          )}

          {creationError && (
            <Alert severity="error" className="text-sm">
              Failed to create cancellation: {creationError.message}
            </Alert>
          )}
        </div>
      </Modal.Body>

      <Modal.Footer className="flex gap-2">
        <div className="grow">
          <FooterMessage count={countRes.data} />
        </div>

        <Button
          appearance="outlined"
          disabled={isCreating}
          kind="secondary"
          label="Close"
          onClick={onClose}
        />
        <Button
          disabled={isCreating || !timeRange || (countRes.data ?? 0) === 0}
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
    return <span>Approximately 1 run will cancel</span>;
  }

  return <span>Approximately {count} runs will cancel</span>;
}
