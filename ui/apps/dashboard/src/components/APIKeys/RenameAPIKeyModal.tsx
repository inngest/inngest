import { useEffect, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Modal } from '@inngest/components/Modal';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { apiKeyErrorMessage } from './errorMessage';
import { validateAPIKeyName } from './validation';

const Mutation = graphql(`
  mutation UpdateAPIKey($input: UpdateAPIKeyInput!) {
    updateAPIKey(input: $input) {
      id
      name
    }
  }
`);

type Props = {
  isOpen: boolean;
  onClose: () => void;
  keyID: string | undefined;
  currentName: string | undefined;
};

export function RenameAPIKeyModal({
  isOpen,
  onClose,
  keyID,
  currentName,
}: Props) {
  const [name, setName] = useState(currentName ?? '');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [, update] = useMutation(Mutation);

  useEffect(() => {
    if (isOpen) {
      setName(currentName ?? '');
      setError(null);
    }
  }, [currentName, isOpen]);

  async function submit() {
    if (!keyID) return;
    setError(null);
    const nameErr = validateAPIKeyName(name);
    if (nameErr) {
      setError(nameErr);
      return;
    }
    const trimmed = name.trim();

    setIsSubmitting(true);
    try {
      const res = await update(
        { input: { id: keyID, name: trimmed } },
        { additionalTypenames: ['APIKey'] },
      );
      if (res.error) {
        setError(apiKeyErrorMessage(res.error, 'Could not rename API key.'));
        return;
      }
      toast.success('API key renamed');
      onClose();
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <Modal className="w-full max-w-xl" isOpen={isOpen} onClose={onClose}>
      <Modal.Header>Rename API key</Modal.Header>
      <Modal.Body>
        <div className="flex flex-col gap-2">
          <label
            htmlFor="api-key-rename"
            className="text-basis text-sm font-medium"
          >
            API key name
          </label>
          <Input
            id="api-key-rename"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={isSubmitting}
          />
          {error && <Alert severity="error">{error}</Alert>}
        </div>
      </Modal.Body>
      <Modal.Footer>
        <div className="flex justify-end gap-2">
          <Button
            appearance="outlined"
            kind="secondary"
            label="Cancel"
            onClick={onClose}
            disabled={isSubmitting}
          />
          <Button
            kind="primary"
            label="Save"
            onClick={submit}
            loading={isSubmitting}
            disabled={isSubmitting}
          />
        </div>
      </Modal.Footer>
    </Modal>
  );
}
