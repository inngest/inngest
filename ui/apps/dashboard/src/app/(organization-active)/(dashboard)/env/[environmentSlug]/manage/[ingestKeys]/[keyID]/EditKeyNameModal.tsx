'use client';

import { useContext, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Modal } from '@inngest/components/Modal';
import { toast } from 'sonner';

import useManagePageTerminology from '../useManagePageTerminology';
import { Context } from './Context';

type EditKeyNameProps = {
  isOpen: boolean;
  onClose: () => void;
  keyID: string;
  keyName: string | null;
};

export default function EditKeyModal({ keyID, keyName, isOpen, onClose }: EditKeyNameProps) {
  const [inputValue, setInputValue] = useState(keyName ?? '');
  const [isDisabled, setDisabled] = useState(true);
  const { save, fetching } = useContext(Context);
  const router = useRouter();
  const currentContent = useManagePageTerminology();

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    setInputValue(e.target.value);
    if (e.target.value !== keyName) {
      setDisabled(false);
    } else {
      setDisabled(true);
    }
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!isDisabled) {
      save({ id: keyID, name: inputValue }).then((result) => {
        if (result.error) {
          toast.error(`${currentContent?.name} has not been updated`);
        } else {
          toast.success(`${currentContent?.name} was successfully updated`);
          router.refresh();
        }
      });
    }
  }

  return (
    <Modal
      isOpen={isOpen}
      className={'w-1/4'}
      onClose={onClose}
      title={`Edit the ${currentContent?.name} Name`}
      footer={
        <div className="flex justify-end gap-2">
          <Button
            appearance="outlined"
            label="Cancel"
            kind="secondary"
            onClick={() => {
              onClose();
            }}
          />
          <Button
            kind="primary"
            label="Save"
            loading={fetching}
            onClick={(e) => {
              handleSubmit(e);
              onClose();
            }}
            disabled={!inputValue}
          />
        </div>
      }
    >
      <div className="p-6">
        <Input
          name="keyName"
          placeholder={`${currentContent?.name} Name`}
          value={inputValue}
          onChange={handleChange}
        />
      </div>
    </Modal>
  );
}
