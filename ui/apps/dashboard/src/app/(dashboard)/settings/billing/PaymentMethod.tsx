'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { capitalCase } from 'change-case';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import Button from '@/components/Button';
import Input from '@/components/Forms/Input';
import { graphql } from '@/gql';
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
        <>
          <Button variant="text" className="font-semibold" onClick={() => setIsEditing(true)}>
            Edit
          </Button>
        </>
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
        <p className="text-sm font-medium text-slate-600">
          Please select a plan above to add a payment method
        </p>
      )}
      {isEditing && <UpdateCardModal onSuccess={onSuccess} onCancel={() => setIsEditing(false)} />}
    </BillingCard>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="mt-1.5 grid grid-cols-2 items-center gap-5 text-sm leading-8 text-slate-600">
      <div className="font-medium">{label}</div>
      <div className="font-bold">{value}</div>
    </div>
  );
}
