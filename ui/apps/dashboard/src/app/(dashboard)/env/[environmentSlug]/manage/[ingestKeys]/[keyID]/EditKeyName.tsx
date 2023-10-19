'use client';

import { useContext, useState } from 'react';
import { useRouter } from 'next/navigation';
import { RocketLaunchIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import { toast } from 'sonner';

import Input from '@/components/Forms/Input';
import useManagePageTerminology from './../useManagePageTerminology';
import { Context } from './Context';

type EditKeyNameProps = {
  keyID: string;
  keyName: string | null;
};

export default function EditKeyButton({ keyID, keyName }: EditKeyNameProps) {
  const [inputValue, setInputValue] = useState(keyName ?? '');
  const [isDisabled, setDisabled] = useState(true);
  const { save } = useContext(Context);
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
    <form className="flex gap-1 text-slate-700" onSubmit={handleSubmit}>
      <Input
        name="keyName"
        placeholder={`${currentContent?.name} Name`}
        value={inputValue}
        onChange={handleChange}
      />
      <Button
        type="submit"
        icon={<RocketLaunchIcon />}
        kind="primary"
        disabled={isDisabled}
        label="Save Name"
      />
    </form>
  );
}
