import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Modal } from '@inngest/components/Modal';
import useDebounce from '@inngest/components/hooks/useDebounce';
import { toast } from 'sonner';

import { useCreateAppMutation } from '@/store/generated';
import isValidUrl from '@/utils/urlValidation';

type AddAppModalProps = {
  isOpen: boolean;
  onClose: () => void;
};

export default function AddAppModal({ isOpen, onClose }: AddAppModalProps) {
  const [inputUrl, setInputUrl] = useState('');
  const [isUrlInvalid, setUrlInvalid] = useState(false);
  const [isDisabled, setDisabled] = useState(true);
  const [_createApp] = useCreateAppMutation();

  const debouncedRequest = useDebounce(() => {
    if (isValidUrl(inputUrl)) {
      setUrlInvalid(false);
    } else {
      setUrlInvalid(true);
    }
  });

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    setInputUrl(e.target.value);
    debouncedRequest();
    if (e.target.value.length > 0) {
      setDisabled(false);
    } else {
      setDisabled(true);
    }
  }

  async function createApp() {
    try {
      const response = await _createApp({
        input: {
          url: inputUrl,
        },
      });
      toast.success('The app was successfully added.');
      console.log('Created app:', response);
    } catch (error) {
      toast.error('The app could not be created: ${error}.');
      console.error('Error creating app:', error);
    }
    onClose();
  }

  function handleSubmit(e: React.SyntheticEvent) {
    e.preventDefault();
    createApp();
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleSubmit(e);
    }
  }

  return (
    <Modal isOpen={isOpen} onClose={onClose} className="min-w-[500px]">
      <Modal.Header description="Sync your Inngest application to the Dev Server">
        Sync App
      </Modal.Header>
      <Modal.Body>
        <form id="add-app" onSubmit={handleSubmit}>
          <div>
            <label htmlFor="addAppUrlModal" className="text-basis text-sm">
              Insert the URL of your application:
            </label>
            <Input
              className="mt-2 w-full"
              id="addAppUrlModal"
              value={inputUrl}
              placeholder="http://localhost:3000/api/inngest"
              onChange={handleChange}
              onKeyDown={handleKeyDown}
              error={isUrlInvalid && inputUrl.length > 0 ? 'Please enter a valid URL' : undefined}
            />
          </div>
        </form>
      </Modal.Body>
      <Modal.Footer className="flex justify-end gap-2">
        <Button label="Cancel" kind="secondary" appearance="outlined" onClick={onClose} />
        <Button
          disabled={isDisabled || isUrlInvalid}
          label="Sync App"
          type="submit"
          form="add-app"
        />
      </Modal.Footer>
    </Modal>
  );
}
