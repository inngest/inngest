import { useState } from 'react';
import Modal from '@/components/Modal';
import Button from '@/components/Button';
import { IconExclamationTriangle } from '@/icons';
import classNames from '@/utils/classnames';
import useInputUrlValidation from '@/hooks/useInputURLValidation';

export default function AddAppModal({ isOpen, onClose }) {
  const [inputUrl, setInputUrl, isUrlInvalid] = useInputUrlValidation();
  const [isDisabled, setDisabled] = useState(true);

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    setInputUrl(e.target.value);
    if (e.target.value.length > 0) {
      setDisabled(false);
    } else {
      setDisabled(true);
    }
  }

  function handleSubmit() {
    // To do: call Add App.
  }

  return (
    <Modal
      title="Add Inngest App"
      description="Connect your Inngest application to the Dev Server"
      isOpen={isOpen}
      onClose={onClose}
    >
      <form>
        <div className="bg-[#050911]/50 p-6">
          <label htmlFor="addAppUrlModal" className="text-sm font-semibold text-white">
            App URL
            <span className="text-slate-500 text-sm pb-4 block">
              The URL of your application
            </span>
          </label>
          <div className="relative">
            <input
              id="addAppUrlModal"
              className={classNames(
                'min-w-[420px] bg-slate-800 rounded-md text-slate-300 py-2 px-4 outline-2 outline-indigo-500 focus:outline',
                isUrlInvalid && 'pr-8 outline-rose-500'
              )}
              placeholder="https://example.com/api/inngest"
              value={inputUrl}
              onChange={handleChange}
            />
            {isUrlInvalid && (
              <IconExclamationTriangle className="absolute top-2/4 right-2 -translate-y-2/4 text-rose-500" />
            )}
          </div>
        </div>
        {isUrlInvalid && (
          <p className="bg-rose-600/50 text-white flex items-center gap-2 text-sm px-6 py-2">
            <IconExclamationTriangle />
            Please enter a valid URL
          </p>
        )}
        <div className="flex items-center justify-between p-6 border-t border-slate-800">
          <Button label="Cancel" kind="secondary" btnAction={onClose} />
          <Button
            disabled={isDisabled || isUrlInvalid}
            label="Connect App"
            btnAction={handleSubmit}
          />
        </div>
      </form>
    </Modal>
  );
}
