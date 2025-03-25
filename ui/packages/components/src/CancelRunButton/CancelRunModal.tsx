'use client';

import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { AlertModal } from '@inngest/components/Modal';
import { toast } from 'sonner';

type Props = {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: () => Promise<void>;
};

export function CancelRunModal({ isOpen, onClose, onSubmit }: Props) {
  const [error, setError] = useState<string>();
  const [isLoading, setIsLoading] = useState(false);

  async function handleSubmit() {
    setIsLoading(true);
    try {
      await onSubmit();
      toast.success('Function run cancelled');
      setError(undefined);
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
    <AlertModal
      className="w-1/3"
      isLoading={isLoading}
      isOpen={isOpen}
      onClose={onClose}
      onSubmit={handleSubmit}
      title="Are you sure you want to cancel this function run?"
    >
      <p className="text-basis px-6 pb-0 pt-4">
        The function run will end early and its status will be "cancelled". This action cannot be
        undone.
      </p>

      {error && (
        <Alert className="mt-6" severity="error">
          {error}
        </Alert>
      )}
    </AlertModal>
  );
}
