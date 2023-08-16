import { useState } from 'react';
import { toast } from 'sonner';

import Button from '@/components/Button';
import Input from '@/components/Form/Input';
import Modal from '@/components/Modal';
import useDebounce from '@/hooks/useDebounce';
import { IconExclamationTriangle } from '@/icons';
import { useCreateAppMutation } from '@/store/generated';
import isValidUrl from '@/utils/urlValidation';

export default function AddAppModal({ isOpen, onClose }) {
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
    // To do: add optimistic render in the list
  }

  function handleSubmit(e) {
    e.preventDefault();
    createApp();
  }

  function handleKeyDown(e) {
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
    >
      <form onSubmit={handleSubmit}>
        <div className="bg-[#050911]/50 p-6">
          <label htmlFor="addAppUrlModal" className="text-sm font-semibold text-white">
            App URL
            <span className="text-slate-500 text-sm pb-4 block">The URL of your application</span>
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
          <p className="bg-rose-600/50 text-white flex items-center gap-2 text-sm px-6 py-2">
            <IconExclamationTriangle />
            Please enter a valid URL
          </p>
        )}
        <div className="flex items-center justify-between p-6 border-t border-slate-800">
          <Button label="Cancel" kind="secondary" btnAction={onClose} />
          <Button disabled={isDisabled || isUrlInvalid} label="Connect App" type="submit" />
        </div>
      </form>
    </Modal>
  );
}
