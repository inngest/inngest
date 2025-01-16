'use client';

import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal/Modal';
import { capitalCase } from 'change-case';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { type CheckoutItem } from './CheckoutModal';

type ConfirmPlanChangeModalProps = {
  action: 'upgrade' | 'downgrade' | 'cancel';
  items: CheckoutItem[];
  onCancel: () => void;
  onSuccess: () => void;
};

const UpdatePlanDocument = graphql(`
  mutation UpdatePlan($planID: ID!) {
    updatePlan(to: $planID) {
      plan {
        id
        name
      }
    }
  }
`);

// Note - this is currently only used for downgrades as Stripe's Payment Intents
// does not allow for saving a default payment method by default when creating
// a subscription. This is also OK as it ensure that we always have a correct
// and updated Credit Card on file when creating a new subscription.
export default function ConfirmPlanChangeModal({
  action,
  items,
  onCancel,
  onSuccess,
}: ConfirmPlanChangeModalProps) {
  const [uiError, setUiError] = useState('');
  const [{ error: apiError }, updatePlan] = useMutation(UpdatePlanDocument);
  const error = apiError?.message || uiError;

  const handlePlanChange = async () => {
    // NOTE - We only support one Stripe plan/product in the API currently
    // so we just grab the first item
    const planID = items[0]?.planID;
    if (!planID) {
      return setUiError('Unable to change your plan - Invalid Plan ID. Please contact support.');
    }
    await updatePlan({ planID });
    onSuccess();
  };

  const amount = items.reduce((total, item) => {
    return total + item.amount * item.quantity;
  }, 0);
  const planName = items.map((item) => item.name).join(', ');
  const isCancellation = action === 'cancel';

  return (
    <Modal className="flex min-w-[600px] max-w-xl flex-col gap-4" isOpen={true} onClose={onCancel}>
      <Modal.Header>
        {isCancellation ? (
          <>Cancel Your Subscription</>
        ) : (
          <>
            {capitalCase(action)} to {planName}
          </>
        )}
      </Modal.Header>

      <Modal.Body>
        {isCancellation ? (
          <>
            <p className="mb-2">
              <b>This is an immediate action.</b> Please confirm before canceling your plan.{' '}
            </p>
            <ul className="list-inside list-disc">
              <li>
                Once canceled, you will lose access to your current plan and its features{' '}
                <b>immediately</b>.
              </li>
              <li>
                You will be credited for any unused time from your current plan, calculated on a
                prorated basis.
              </li>
            </ul>
          </>
        ) : action === 'downgrade' ? (
          <>
            <p className="mb-2">
              <b>This is an immediate action.</b> Please confirm before downgrading your plan.{' '}
            </p>
            <ul className="list-inside list-disc">
              <li>
                Once donwgraded, you will lose access to your current plan and its features{' '}
                <b>immediately</b>.
              </li>
              <li>
                You will be credited for any unused time from your current plan, calculated on a
                prorated basis.
              </li>
            </ul>
          </>
        ) : (
          <p>You have chosen wisely - Please confirm your upgrade!</p>
        )}
        <p className="my-4 font-semibold">New monthly cost: ${amount / 100}</p>
        <div className="mt-6 flex flex-row justify-end">
          <Button
            kind={isCancellation || action === 'downgrade' ? 'danger' : 'primary'}
            onClick={handlePlanChange}
            label={`Confirm ${capitalCase(action)}`}
          />
        </div>
        {Boolean(error) && <Alert severity="error">{error}</Alert>}
      </Modal.Body>
    </Modal>
  );
}
