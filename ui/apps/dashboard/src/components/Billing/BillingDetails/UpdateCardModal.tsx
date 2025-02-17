'use client';

import { useState } from 'react';
import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal/Modal';
import { CardElement, Elements, useElements, useStripe } from '@stripe/react-stripe-js';
import { loadStripe } from '@stripe/stripe-js';
import { useMutation } from 'urql';

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
      <Modal.Header>Update your payment method</Modal.Header>
      <Modal.Body>
        <Elements
          stripe={stripePromise}
          options={{
            mode: 'setup',
            currency: 'usd',
          }}
        >
          <CheckoutForm onSuccess={onSuccess} />
        </Elements>
      </Modal.Body>
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
          onClick={handleSubmit}
          kind="primary"
          label="Change Payment Method"
        />
      </div>
      {Boolean(error) && (
        <Alert severity="error" className="my-4 text-sm">
          {error}
        </Alert>
      )}
    </form>
  );
}
