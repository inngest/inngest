'use client';

import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import { Modal } from '@inngest/components/Modal';
import { toast } from 'sonner';

import Input from '@/components/Forms/Input';
import { TimeInput } from '@/components/TimeRangeInput/TimeInput';

type Props = {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (args: {
    expression: string | undefined;
    name: string | undefined;
    queuedAtMax: Date;
    queuedAtMin: Date | undefined;
  }) => Promise<void>;
};

export function CreateCancellationModal(props: Props) {
  const { isOpen } = props;
  const [error, setError] = useState<string>();
  const [isLoading, setIsLoading] = useState(false);
  const [expression, setExpression] = useState<string>();
  const [name, setName] = useState<string>();
  const [queuedAtMax, setQueuedAtMax] = useState<Date>();
  const [queuedAtMin, setQueuedAtMin] = useState<Date>();

  function onClose() {
    props.onClose();
    setError(undefined);
    setExpression(undefined);
    setName(undefined);
    setQueuedAtMin(undefined);
    setQueuedAtMax(undefined);
  }

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsLoading(true);
    try {
      if (!queuedAtMax) {
        throw new Error('Queued before time is required');
      }
      if (queuedAtMin && queuedAtMin > queuedAtMax) {
        throw new Error('Queued after time must be before queued before time');
      }

      await props.onSubmit({ expression, name, queuedAtMax, queuedAtMin });
      toast.success('Created cancellation');
      setError(undefined);
      setExpression(undefined);
      setName(undefined);
      setQueuedAtMin(undefined);
      setQueuedAtMax(undefined);
    } catch (error) {
      if (!(error instanceof Error)) {
        setError('Unknown error');
        return;
      }

      setError(error.message);
      throw error;
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <Modal className="w-full max-w-3xl" isOpen={isOpen} onClose={onClose}>
      <Modal.Header description="Cancel multiple function runs in bulk">
        Create Cancellation
      </Modal.Header>

      <form onSubmit={onSubmit}>
        <Modal.Body className="m-0">
          <Alert className="rounded-none" severity="info">
            Cancellation may take a few minutes to complete. In the meantime, matching runs may
            still have the {'"Running"'} status but they will eventually cancel.
          </Alert>

          <div className="divide-y divide-slate-100">
            <div className="flex items-start p-6">
              <label
                htmlFor="cancellation-name"
                className="flex-1 text-sm font-medium text-slate-700"
              >
                Cancellation Name
                <p className="text-xs text-slate-500">
                  Specify the name to identify the cancellation on the dashboard
                </p>
              </label>

              <div className="flex-1">
                <Input
                  disablePasswordManager
                  maxLength={64}
                  name="cancellation-name"
                  onChange={(event) => setName(event.target.value)}
                  type="text"
                />
              </div>
            </div>

            <div>
              <div className="flex items-start p-6">
                <label
                  htmlFor="queued-at-min"
                  className="flex-1 text-sm font-medium text-slate-700"
                >
                  Queued Time Minimum
                  <p className="text-xs text-slate-500">Include runs queued after this time</p>
                </label>

                <div className="flex-1">
                  <TimeInput name="queued-at-min" onChange={(date) => setQueuedAtMin(date)} />
                </div>
              </div>

              <div className="flex items-start p-6">
                <label
                  htmlFor="queued-at-max"
                  className="flex-1 text-sm font-medium text-slate-700"
                >
                  Queued Time Maximum (Required)
                  <p className="text-xs text-slate-500">Include runs queued before this time</p>
                </label>

                <div className="flex-1">
                  <TimeInput
                    name="queued-at-max"
                    onChange={(date) => setQueuedAtMax(date)}
                    placeholder="now"
                  />
                </div>
              </div>
            </div>

            <div className="p-6">
              <label htmlFor="expression" className="text-sm font-medium text-slate-700">
                Expression
                <p className="text-xs text-slate-500">
                  Include runs whose event matches a{' '}
                  <Link
                    className="inline-flex"
                    internalNavigation={false}
                    href="https://www.inngest.com/docs/apps"
                  >
                    CEL expression
                  </Link>
                  .
                </p>
              </label>

              <div className="mt-4">
                <Input
                  maxLength={1024}
                  minLength={3}
                  name="expression"
                  onChange={(event) => setExpression(event.target.value)}
                  placeholder="event.data.name == 'example'"
                  type="text"
                />
              </div>
            </div>
          </div>
        </Modal.Body>

        <Modal.Footer>
          {error && (
            <Alert className="mb-6" severity="error">
              {error}
            </Alert>
          )}

          <div className="flex justify-end gap-2">
            <Button appearance="outlined" btnAction={onClose} disabled={isLoading} label="Close" />
            <Button
              appearance="solid"
              disabled={isLoading}
              kind="danger"
              label="Submit"
              loading={isLoading}
              type="submit"
            />
          </div>
        </Modal.Footer>
      </form>
    </Modal>
  );
}
