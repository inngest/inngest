import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal/Modal';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';
import {
  Elements,
  PaymentElement,
  useElements,
  useStripe,
} from '@stripe/react-stripe-js';
import { loadStripe } from '@stripe/stripe-js';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { type StripeSubscriptionItemsInput } from '@/gql/graphql';
import {
  backgroundColor,
  colors,
  textColor,
  placeholderColor,
} from '@/utils/tailwind';

export type CheckoutItem = {
  /* Inngest plan slug */
  planSlug: string;
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
const stripePromise = loadStripe(
  import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY || '',
);

export default function CheckoutModal({
  items,
  onCancel,
  onSuccess,
}: CheckoutModalProps) {
  const amount = items.reduce((total, item) => {
    return total + item.amount * item.quantity;
  }, 0);
  const planName = items.map((item) => item.name).join(', ');
  return (
    <Modal
      className="flex min-w-[600px] max-w-xl flex-col gap-4"
      isOpen={true}
      onClose={onCancel}
    >
      <Modal.Header>Upgrade to {planName}</Modal.Header>

      <Modal.Body>
        <Elements
          stripe={stripePromise}
          options={{
            mode: 'subscription',
            amount: amount,
            currency: 'usd',
            appearance: {
              variables: {
                colorText: resolveColor(textColor.basis, isDark()),
                colorPrimary: resolveColor(colors.primary.moderate, isDark()),
                colorBackground: resolveColor(
                  backgroundColor.canvasBase,
                  isDark(),
                ),
                colorTextSecondary: resolveColor(textColor.subtle, isDark()),
                colorDanger: resolveColor(textColor.error, isDark()),
                colorWarning: resolveColor(textColor.warning, isDark()),
                colorTextPlaceholder: resolveColor(
                  placeholderColor.disabled,
                  isDark(),
                ),
              },
            },
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
      subscriptionId
    }
  }
`);

const ConfirmSubscriptionUpgradeDocument = graphql(`
  mutation ConfirmSubscriptionUpgrade($subscriptionId: String!) {
    confirmSubscriptionUpgrade(subscriptionId: $subscriptionId) {
      success
      message
      account {
        id
      }
    }
  }
`);

function CheckoutForm({
  items,
  onSuccess,
}: {
  items: CheckoutItem[];
  onSuccess: () => void;
}) {
  const stripe = useStripe();
  const elements = useElements();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const [, createStripeSubscription] = useMutation(
    CreateStripeSubscriptionDocument,
  );
  const [, confirmSubscriptionUpgrade] = useMutation(
    ConfirmSubscriptionUpgradeDocument,
  );

  const handleSubmit = async (e: React.SyntheticEvent) => {
    e.preventDefault();
    // This should not happen since the button is disabled before this
    if (!stripe || !elements) {
      console.error('Stripe Elements not loaded');
      return;
    }

    setLoading(true);
    setError('');

    const { error: submitError } = await elements.submit();
    if (submitError) {
      setLoading(false);
      return setError(
        submitError.message ||
          'Sorry, there was an issue saving your payment information',
      );
    }

    const apiItems = items.map(({ planSlug, quantity, amount }) => ({
      planSlug,
      quantity,
      amount,
    }));

    // Create the subscription
    const { data, error: createSubscriptionError } =
      await createStripeSubscription({
        input: { items: apiItems },
      });
    if (createSubscriptionError) {
      setLoading(false);
      return setError(
        createSubscriptionError.message ||
          'Sorry, there was an issue creating your subscription',
      );
    }

    const clientSecret = data?.createStripeSubscription.clientSecret || '';
    const subscriptionId = data?.createStripeSubscription.subscriptionId || '';
    const message = data?.createStripeSubscription.message || '';

    // Plan change on existing subscription - already updated
    if (message === 'Updated subscription') {
      setLoading(false);
      return onSuccess();
    }

    // No payment needed, just confirm the upgrade
    if (!clientSecret && subscriptionId) {
      const { data: confirmData, error: confirmError } =
        await confirmSubscriptionUpgrade({ subscriptionId });
      if (confirmError || !confirmData?.confirmSubscriptionUpgrade.success) {
        setLoading(false);
        return setError(
          confirmError?.message ||
            confirmData?.confirmSubscriptionUpgrade.message ||
            'Sorry, there was an issue confirming your subscription upgrade',
        );
      }
      setLoading(false);
      return onSuccess();
    }

    if (!clientSecret) {
      setLoading(false);
      return setError('Sorry, there was an issue creating your subscription');
    }

    // Confirm the payment with Stripe
    const { error: stripeConfirmPaymentError } = await stripe.confirmPayment({
      elements,
      clientSecret,
      confirmParams: {
        // TODO - Use PUBLIC_APP_URL from other branch changes
        return_url: new URL(
          '/account/billing',
          import.meta.env.VITE_APP_URL,
        ).toString(),
      },
      redirect: 'if_required',
    });

    if (stripeConfirmPaymentError) {
      setLoading(false);
      return setError(
        stripeConfirmPaymentError.message ||
          'Sorry, there was an issue confirming your payment',
      );
    }

    // Confirm the subscription upgrade in our backend
    const { data: confirmData, error: confirmError } =
      await confirmSubscriptionUpgrade({ subscriptionId });

    if (confirmError || !confirmData?.confirmSubscriptionUpgrade.success) {
      setLoading(false);
      console.error('Payment succeeded but subscription confirmation failed:', {
        subscriptionId,
        error: confirmError || confirmData?.confirmSubscriptionUpgrade.message,
      });
      return setError(
        'Your payment was successful, but we encountered an issue activating your subscription. ' +
          'Please contact support at hello@inngest.com with your subscription ID: ' +
          subscriptionId,
      );
    }

    setLoading(false);
    onSuccess();
  };

  return (
    <form onSubmit={handleSubmit}>
      <div className="mb-2 min-h-[290px]">
        <PaymentElement />
      </div>
      {error ? (
        <Alert severity="error" className="text-sm">
          {error}
        </Alert>
      ) : (
        <Alert severity="info" className="text-sm">
          <p>Subscriptions are billed on the 1st of each month.</p>
          <ul className="list-inside list-disc">
            <li>
              When upgrading, you will be charged a prorated amount for the
              remaining days of the month based on the new plan.
            </li>
            <li>
              If you switch from one paid plan to another, you will be credited
              for any unused time from your previous plan, calculated on a
              prorated basis.
            </li>
            <li>
              Additional usage is calculated and billed at the end of the month.
            </li>
          </ul>
        </Alert>
      )}
      <div className="mt-6 flex flex-row justify-end">
        <Button
          type="submit"
          className="px-16"
          disabled={!stripe || loading}
          onClick={handleSubmit}
          label="Complete Upgrade"
        />
      </div>
    </form>
  );
}
