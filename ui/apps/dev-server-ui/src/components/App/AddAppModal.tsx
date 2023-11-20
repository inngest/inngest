import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';
import { toast } from 'sonner';

import Input from '@/components/Form/Input';
import useDebounce from '@/hooks/useDebounce';
import { IconExclamationTriangle } from '@/icons';
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
  const [_createApp, createAppState] = useCreateAppMutation();

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
    <Modal
      title="Add Inngest App"
      description="Connect your Inngest application to the Dev Server"
      isOpen={isOpen}
      onClose={onClose}
      footer={
        <div className="flex items-center justify-between">
          <Button label="Cancel" appearance="outlined" btnAction={onClose} />
          <Button
            disabled={isDisabled || isUrlInvalid}
            label="Connect App"
            type="submit"
            form="add-app"
          />
        </div>
      }
    >
      <form id="add-app" onSubmit={handleSubmit}>
        <div className="p-6">
          <label htmlFor="addAppUrlModal" className="text-sm font-semibold text-white">
            App URL
            <span className="block pb-4 text-sm text-slate-500">The URL of your application</span>
          </label>
          <Input
            id="addAppUrlModal"
            value={inputUrl}
            placeholder="http://localhost:3000/api/inngest"
            onChange={handleChange}
            onKeyDown={handleKeyDown}
            isInvalid={isUrlInvalid}
          />
        </div>
        {isUrlInvalid && inputUrl.length > 0 && (
          <p className="flex items-center gap-2 bg-rose-600/50 px-6 py-2 text-sm text-white">
            <IconExclamationTriangle />
            Please enter a valid URL
          </p>
        )}
      </form>
    </Modal>
  );
}
