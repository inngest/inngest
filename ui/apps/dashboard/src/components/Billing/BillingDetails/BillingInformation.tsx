'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import BillingCard from './BillingCard';

const updateBillingInformationDocument = graphql(`
  mutation UpdateAccount($input: UpdateAccount!) {
    account: updateAccount(input: $input) {
      billingEmail
      name
    }
  }
`);

export default function BillingInformation({
  billingEmail,
  accountName,
}: {
  billingEmail: string;
  accountName: string | null | undefined;
}) {
  const [isEditing, setIsEditing] = useState(false);
  const [email, setEmail] = useState(billingEmail || '');
  const [name, setName] = useState(accountName || '');
  const isSaveDisabled = email === billingEmail && name === accountName;
  const [, updateBillingInformation] = useMutation(updateBillingInformationDocument);
  const router = useRouter();

  function onEditButtonClick() {
    setIsEditing(true);
  }

  function handleEmailChange(e: React.ChangeEvent<HTMLInputElement>) {
    setEmail(e.target.value);
  }

  function handleNameChange(e: React.ChangeEvent<HTMLInputElement>) {
    setName(e.target.value);
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (isSaveDisabled) return;

    updateBillingInformation({ input: { billingEmail: email, name } }).then((result) => {
      if (result.error) {
        toast.error(`Billing information has not been updated: ${result.error.message}`);
      } else {
        toast.success(`Billing information was successfully updated`);
        router.refresh();
      }
    });
    setIsEditing(false);
  }

  return (
    <BillingCard
      heading="Billing Information"
      className="mb-3"
      actions={
        <>
          {isEditing ? (
            <Button
              appearance="ghost"
              disabled={isSaveDisabled}
              className="h-6 font-semibold"
              type="submit"
              onClick={handleSubmit}
              label="Save"
            />
          ) : (
            <Button
              appearance="ghost"
              kind="primary"
              className="h-6 font-semibold"
              onClick={onEditButtonClick}
              label="Edit"
            />
          )}
        </>
      }
    >
      {billingEmail || name ? (
        <>
          <div className="text-subtle mt-1.5 grid grid-cols-2 items-center gap-5 text-sm">
            <label htmlFor="name" className="font-medium">
              Name
            </label>
            <Input
              name="name"
              placeholder="Name"
              defaultValue={name}
              type="text"
              onChange={handleNameChange}
              readOnly={!isEditing}
            />
          </div>
          <div className="text-subtle mt-1.5 grid grid-cols-2 items-center gap-5 text-sm">
            <label htmlFor="billingEmail" className="font-medium">
              Billing Email
            </label>
            <Input
              name="billingEmail"
              placeholder="Email address"
              defaultValue={email}
              type="email"
              onChange={handleEmailChange}
              readOnly={!isEditing}
            />
          </div>
        </>
      ) : (
        <p className="text-subtle text-sm font-medium">
          Please select a plan to add billing information
        </p>
      )}
    </BillingCard>
  );
}
