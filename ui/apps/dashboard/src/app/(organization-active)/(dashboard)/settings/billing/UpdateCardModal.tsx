'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { RiBankCardLine, RiErrorWarningLine } from '@remixicon/react';
import { CardElement, Elements, useElements, useStripe } from '@stripe/react-stripe-js';
import { loadStripe } from '@stripe/stripe-js';
import { useMutation } from 'urql';

import Modal from '@/components/Modal';
import { graphql } from '@/gql';

type CheckoutModalProps = {
  onCancel: () => void;
  onSuccess: () => void;
};

// Make sure to call `loadStripe` outside of a component’s render to avoid
// recreating the `Stripe` object on every render.
const stripePromise = loadStripe(process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY || '');

export default function CheckoutModal({ onCancel, onSuccess }: CheckoutModalProps) {
  return (
    <Modal className="flex min-w-[600px] max-w-xl flex-col gap-4" isOpen={true} onClose={onCancel}>
      <header className="flex flex-row items-center gap-3">
        <RiBankCardLine className="h-5 text-indigo-500" />
        <h2 className="text-lg font-semibold">Update your payment method</h2>
      </header>

      <div>
        <Elements
          stripe={stripePromise}
          options={{
            mode: 'setup',
            currency: 'usd',
          }}
        >
          <CheckoutForm onSuccess={onSuccess} />
        </Elements>
      </div>
    </Modal>
  );
}

const UpdatePaymentMethodDocument = graphql(`
  mutation UpdatePaymentMethod($token: String!) {
    updatePaymentMethod(token: $token) {
      brand
      last4
      expMonth
      expYear
      createdAt
      default
    }
  }
`);

function CheckoutForm({ onSuccess }: { onSuccess: () => void }) {
  const stripe = useStripe();
  const elements = useElements();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const [, updatePaymentMethod] = useMutation(UpdatePaymentMethodDocument);

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
      setLoading(false);
      return setError(
        submitError.message || 'Sorry, there was an issue saving your payment information'
      );
    }
    let token;
    try {
      const result = await stripe.createToken(elements.getElement('card') as any);
      if (result.error) {
        setLoading(false);
        return setError(
          result.error.message || 'Sorry, there was an issue saving your payment information'
        );
      }
      token = result.token.id;
    } catch (err) {
      setError('Sorry, there was an issue confirming your payment');
      setLoading(false);
      return;
    }

    const { error: updateError } = await updatePaymentMethod({ token });
    setLoading(false);

    if (updateError) {
      setError(updateError.message || 'Sorry, there was an issue confirming your payment');
    } else {
      onSuccess();
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <div className="min-h-[50px]">
        <CardElement options={{}} />
      </div>
      <div className="mt-6 flex flex-row justify-end">
        <Button
          type="submit"
          className="px-16"
          disabled={!stripe || loading}
          btnAction={handleSubmit}
          kind="primary"
          label="Change Payment Method"
        />
      </div>
      {/* TODO - Explore re-use alert from signing key page PR */}
      {Boolean(error) && (
        <div className="my-4 flex rounded-md border border-red-600 bg-red-100 p-4 text-sm text-red-600">
          <RiErrorWarningLine className="mr-2 w-4 text-red-600" />
          {error}
        </div>
      )}
    </form>
  );
}
