'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { capitalCase } from 'change-case';

import BillingCard from './BillingCard';
import UpdateCardModal from './UpdateCardModal';

export default function PaymentMethod({
  paymentMethod,
}: {
  paymentMethod: {
    __typename?: 'PaymentMethod';
    brand: string;
    last4: string;
    expMonth: string;
    expYear: string;
    createdAt: string;
    default: boolean;
  } | null;
}) {
  const [isEditing, setIsEditing] = useState(false);
  const router = useRouter();

  const onSuccess = () => {
    setIsEditing(false);
    router.refresh();
  };

  return (
    <BillingCard
      heading="Payment Method"
      className="mb-3"
      actions={
        <Button
          appearance="ghost"
          kind="primary"
          onClick={() => setIsEditing(true)}
          label="Edit"
          className="h-6 font-semibold"
        />
      }
    >
      {paymentMethod ? (
        <>
          <Row
            label="Credit Card"
            value={`${capitalCase(paymentMethod.brand)} ending in ${paymentMethod.last4}`}
          />
          <Row label="Expiration" value={`${paymentMethod.expMonth}/${paymentMethod.expYear}`} />
        </>
      ) : (
        <p className="text-subtle text-sm font-medium">
          Please select a paid plan to add a payment method
        </p>
      )}
      {isEditing && <UpdateCardModal onSuccess={onSuccess} onCancel={() => setIsEditing(false)} />}
    </BillingCard>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="text-subtle mt-1.5 grid grid-cols-2 items-center gap-5 text-sm leading-8">
      <div className="font-medium">{label}</div>
      <div className="font-bold">{value}</div>
    </div>
  );
}
