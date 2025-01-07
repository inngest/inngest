'use client';

import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal/Modal';
import { Elements, PaymentElement, useElements, useStripe } from '@stripe/react-stripe-js';
import { loadStripe } from '@stripe/stripe-js';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { type StripeSubscriptionItemsInput } from '@/gql/graphql';

export type CheckoutItem = {
  /* Inngest plan id */
  planID: string;
  name: string;
  quantity: number;
  amount: number;
} & StripeSubscriptionItemsInput;

type CheckoutModalProps = {
  items: CheckoutItem[];
  onCancel: () => void;
  onSuccess: () => void;
};

// Make sure to call `loadStripe` outside of a component’s render to avoid
// recreating the `Stripe` object on every render.
const stripePromise = loadStripe(process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY || '');

export default function CheckoutModal({ items, onCancel, onSuccess }: CheckoutModalProps) {
  const amount = items.reduce((total, item) => {
    return total + item.amount * item.quantity;
  }, 0);
  const planName = items.map((item) => item.name).join(', ');
  return (
    <Modal className="flex min-w-[600px] max-w-xl flex-col gap-4" isOpen={true} onClose={onCancel}>
      <Modal.Header>Upgrade to {planName}</Modal.Header>

      <Modal.Body>
        <Elements
          stripe={stripePromise}
          options={{
            mode: 'subscription',
            amount: amount,
            currency: 'usd',
          }}
        >
          <CheckoutForm items={items} onSuccess={onSuccess} />
        </Elements>
      </Modal.Body>
    </Modal>
  );
}

const CreateStripeSubscriptionDocument = graphql(`
  mutation CreateStripeSubscription($input: StripeSubscriptionInput!) {
    createStripeSubscription(input: $input) {
      clientSecret
      message
    }
  }
`);

function CheckoutForm({ items, onSuccess }: { items: CheckoutItem[]; onSuccess: () => void }) {
  const stripe = useStripe();
  const elements = useElements();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const [, createStripeSubscription] = useMutation(CreateStripeSubscriptionDocument);

  const handleSubmit = async (e: React.SyntheticEvent) => {
    e.preventDefault();
    // This should not happen since the button is disabled before this
    if (!stripe || !elements) {
      console.error('Stripe Elements not loaded');
      return;
    }

    setLoading(true);

    const { error: submitError } = await elements.submit();
    if (submitError) {
      return setError(
        submitError.message || 'Sorry, there was an issue saving your payment information'
      );
    }

    const apiItems = items.map(({ planID, quantity, amount }) => ({ planID, quantity, amount }));

    // Create the PaymentIntent
    const { data, error: createSubscriptionError } = await createStripeSubscription({
      input: { items: apiItems },
    });
    if (createSubscriptionError) {
      return setError(
        createSubscriptionError.message || 'Sorry, there was an issue changing your subscription'
      );
    }

    const clientSecret = data?.createStripeSubscription.clientSecret || '';
    // If there is no client secret, the payment is already associated with the subscription,
    // we can return success early
    if (!clientSecret) {
      return onSuccess();
    }

    // Confirm the PaymentIntent using the details collected by the Payment Element
    const { error: stripeConfirmPaymentError } = await stripe.confirmPayment({
      elements,
      clientSecret,
      confirmParams: {
        // TODO - Use PUBLIC_APP_URL from other branch changes
        return_url: new URL('/account/billing', process.env.NEXT_PUBLIC_APP_URL).toString(),
      },
      redirect: 'if_required',
    });

    if (stripeConfirmPaymentError) {
      setError(
        stripeConfirmPaymentError.message || 'Sorry, there was an issue confirming your payment'
      );
    } else {
      onSuccess();
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <div className="mb-2 min-h-[290px]">
        <PaymentElement />
      </div>
      <Alert severity="info" className="text-sm">
        <p>Subscriptions are billed on the 1st of each month.</p>
        <ul className="list-inside list-disc">
          <li>
            When upgrading, you will be charged a prorated amount for the remaining days of the
            month based on the new plan.
          </li>
          <li>
            If you switch from one paid plan to another, you will be credited for any unused time
            from your previous plan, calculated on a prorated basis.
          </li>
          <li>Additional usage is calculated and billed at the end of the month.</li>
        </ul>
      </Alert>
      <div className="mt-6 flex flex-row justify-end">
        <Button
          type="submit"
          className="px-16"
          disabled={!stripe || loading}
          onClick={handleSubmit}
          label="Complete Upgrade"
        />
      </div>
      {Boolean(error) && (
        <Alert severity="error" className="text-sm">
          {error}
        </Alert>
      )}
    </form>
  );
}
