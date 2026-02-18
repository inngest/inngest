import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Modal, AlertModal } from '@inngest/components/Modal';
import { useMutation } from 'urql';
import { toast } from 'sonner';

import { EnrollToConstraintAPIMutation, type ConstraintAPIData } from './data';

type ConstraintAPIModalProps = {
  isOpen: boolean;
  onClose: () => void;
  constraintAPIData: ConstraintAPIData;
};

export function ConstraintAPIModal({
  isOpen,
  onClose,
  constraintAPIData,
}: ConstraintAPIModalProps) {
  const [showConfirmation, setShowConfirmation] = useState(false);
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

      toast.success('Successfully enrolled in Constraint API!');
      setShowConfirmation(false);
      onClose();
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Unknown error occurred';
      setError(message);
    } finally {
      setIsFetching(false);
    }
  }

  // Confirmation modal (shown after clicking "Enroll Now")
  if (showConfirmation) {
    return (
      <AlertModal
        isOpen={isOpen}
        onClose={() => {
          setShowConfirmation(false);
          setError(undefined);
        }}
        onSubmit={handleEnrollment}
        title="Confirm Enrollment"
        description="Are you sure you want to enroll in the Constraint API? This will enable the new infrastructure for your account."
        confirmButtonLabel="Enroll Now"
        confirmButtonKind="primary"
        isLoading={isFetching}
      >
        {error && (
          <div className="p-6">
            <Alert severity="error">{error}</Alert>
          </div>
        )}
      </AlertModal>
    );
  }

  // Main information modal
  return (
    <Modal className="max-w-2xl" isOpen={isOpen} onClose={onClose}>
      <Modal.Header>
        {displayState === 'not_enrolled' && 'Constraint API Upgrade'}
        {displayState === 'pending' && 'Enrollment Pending'}
        {displayState === 'active' && 'Constraint API Active'}
      </Modal.Header>

      <Modal.Body>
        {displayState === 'not_enrolled' && (
          <>
            <p className="mb-4">
              We&apos;re rolling out a new infrastructure upgrade that improves
              reliability and performance through our Constraint API.
            </p>
            <Alert severity="info" className="mb-4">
              You can enroll now to start using the new infrastructure when it
              becomes available for your account.
            </Alert>
            <p className="text-sm text-subtle">
              Enrollment is optional but recommended. You can enroll at any time
              that best fits your schedule.
            </p>
          </>
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
              onClick={() => setShowConfirmation(true)}
            />
          )}
        </div>
      </Modal.Footer>
    </Modal>
  );
}
