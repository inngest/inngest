import { useEffect, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Modal } from '@inngest/components/Modal';
import useDebounce from '@inngest/components/hooks/useDebounce';
import { toast } from 'sonner';

import { useCreateAppMutation } from '@/store/generated';
import { useInfoQuery } from '@/store/devApi';
import isValidUrl from '@/utils/urlValidation';

const STORAGE_KEY = 'app:inngest_app_url';

type AddAppModalProps = {
  isOpen: boolean;
  onClose: () => void;
};

export default function AddAppModal({ isOpen, onClose }: AddAppModalProps) {
  const { data: info, isLoading, error } = useInfoQuery();
  const isDevServer = error ? false : !info?.isSingleNodeService;

  const [inputUrl, setInputUrl] = useState('');
  const [isUrlInvalid, setUrlInvalid] = useState(false);
  const [isDisabled, setDisabled] = useState(true);
  const [_createApp] = useCreateAppMutation();

  useEffect(() => {
    const savedUrl = localStorage.getItem(STORAGE_KEY);
    if (savedUrl) {
      setInputUrl(savedUrl);
      setDisabled(false);
      setUrlInvalid(!isValidUrl(savedUrl));
    }
  }, []);

  const debouncedRequest = useDebounce((value: string) => {
    setUrlInvalid(!isValidUrl(value));
  });

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const value = e.target.value;
    setInputUrl(value);

    localStorage.setItem(STORAGE_KEY, value);

    debouncedRequest(value);
    setDisabled(value.length === 0);
  }

  async function createApp() {
    try {
      const response = await _createApp({
        input: { url: inputUrl },
      });

      localStorage.setItem(STORAGE_KEY, inputUrl);

      toast.success('The app was successfully added.');
      console.log('Created app:', response);
    } catch (error) {
      toast.error(`The app could not be created: ${error}`);
      console.error('Error creating app:', error);
      return;
    }

    onClose();
  }

  function handleSubmit(e: React.SyntheticEvent) {
    e.preventDefault();
    if (!isUrlInvalid && inputUrl.length > 0) {
      createApp();
    }
  }

  return (
    <Modal isOpen={isOpen} onClose={onClose} className="min-w-[500px]">
      <Modal.Header
        description={`Sync your Inngest application to the ${
          isDevServer ? 'Dev Server' : 'Server'
        }`}
      >
        Sync App
      </Modal.Header>

      <Modal.Body>
        <form id="add-app" onSubmit={handleSubmit}>
          <label htmlFor="addAppUrlModal" className="text-basis text-sm">
            Insert the URL of your application:
          </label>

          <Input
            className="mt-2 w-full"
            id="addAppUrlModal"
            value={inputUrl}
            placeholder="http://localhost:3000/api/inngest"
            onChange={handleChange}
            error={
              isUrlInvalid && inputUrl.length > 0
                ? 'Please enter a valid URL'
                : undefined
            }
          />
        </form>
      </Modal.Body>

      <Modal.Footer className="flex justify-end gap-2">
        <Button
          label="Cancel"
          kind="secondary"
          appearance="outlined"
          onClick={onClose}
        />
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
