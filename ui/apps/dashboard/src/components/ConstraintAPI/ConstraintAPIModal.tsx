import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';
import { useMutation } from 'urql';

import { EnrollToConstraintAPIMutation, type ConstraintAPIData } from './data';

type ConstraintAPIModalProps = {
  isOpen: boolean;
  onClose: () => void;
  onEnrolled: () => void;
  constraintAPIData: ConstraintAPIData;
};

export function ConstraintAPIModal({
  isOpen,
  onClose,
  onEnrolled,
  constraintAPIData,
}: ConstraintAPIModalProps) {
  const [error, setError] = useState<string>();
  const [isFetching, setIsFetching] = useState(false);
  const [, enrollMutation] = useMutation(EnrollToConstraintAPIMutation);

  const { displayState } = constraintAPIData;

  async function handleEnrollment() {
    setIsFetching(true);
    setError(undefined);

    try {
      const result = await enrollMutation(
        {},
        {
          additionalTypenames: ['Account'], // Cache busting
        },
      );

      if (result.error) {
        throw result.error;
      }

      onClose();
      onEnrolled();
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Unknown error occurred';
      setError(message);
    } finally {
      setIsFetching(false);
    }
  }

  // Main information modal
  return (
    <Modal className="max-w-2xl" isOpen={isOpen} onClose={onClose}>
      <div className="bg-modalBase border-subtle flex items-center justify-between border-b p-6">
        <p className="text-basis text-xl">Important Infrastructure Upgrade</p>
        <a
          href="https://inngest.com/blog"
          target="_blank"
          rel="noopener noreferrer"
          className="text-sm text-link hover:underline"
        >
          Learn more
        </a>
      </div>

      <Modal.Body>
        {displayState === 'not_enrolled' && (
          <div className="space-y-4">
            <div>
              <p className="mb-1 font-medium">Summary</p>
              <p className="text-subtle text-sm">
                We&apos;re upgrading the infrastructure that powers constraint
                enforcement across the platform. As part of this rollout, your
                existing constraint usage will be reset. This only affects
                function execution, <b>billing will not be affected</b>.
              </p>
            </div>
            <div>
              <p className="mb-1 font-medium">Impact</p>
              <ul className="text-subtle list-disc space-y-1 pl-4 text-sm">
                <li>
                  Existing function runs will not count toward your account,
                  function, or custom concurrency limits during the enrollment.
                  As a result, those limits may temporarily be exceeded.
                </li>
                <li>
                  Throttle and rate limit counters on your functions will reset
                  to zero at the time of enrollment.
                </li>
              </ul>
            </div>
            <div>
              <p className="mb-1 font-medium">Mitigation</p>
              <p className="text-subtle mb-2 text-sm">
                If any of your deployed apps may struggle to handle increased
                traffic — whether from higher concurrency or additional
                scheduled runs due to throttle or rate limit resets — you have a
                couple of options:
              </p>
              <ul className="text-subtle list-disc space-y-1 pl-4 text-sm">
                <li>
                  <a
                    href="https://www.inngest.com/docs/guides/pause-functions"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-link hover:underline"
                  >
                    Pause individual functions
                  </a>{' '}
                  before enrolling and re-enable them once the migration is
                  complete.
                </li>
                <li>
                  Reduce your configured concurrency, throttle, or rate limits
                  ahead of time to give yourself more headroom.
                </li>
              </ul>
            </div>
            <div>
              <p className="mb-1 font-medium">Timeline</p>
              <p className="text-subtle text-sm">
                You can enroll at any time before February 23, 2026. After that
                date, any remaining accounts will be migrated automatically.
              </p>
            </div>
          </div>
        )}

        {displayState === 'pending' && (
          <p>
            Your enrollment has been received and is being processed. The
            Constraint API will be activated for your account soon.
          </p>
        )}

        {displayState === 'active' && (
          <>
            <p className="mb-4">
              The Constraint API is now active for your account. Your
              infrastructure is using the upgraded system.
            </p>
            <Alert severity="success">
              ✓ Successfully enrolled and activated
            </Alert>
          </>
        )}
      </Modal.Body>

      <Modal.Footer>
        {error && (
          <Alert severity="error" className="mb-4">
            {error}
          </Alert>
        )}
        <div className="flex justify-end gap-2">
          <Button
            label="Close"
            appearance="outlined"
            kind="secondary"
            onClick={onClose}
          />
          {displayState === 'not_enrolled' && (
            <Button
              label="Enroll Now"
              kind="primary"
              loading={isFetching}
              onClick={handleEnrollment}
            />
          )}
        </div>
      </Modal.Footer>
    </Modal>
  );
}
