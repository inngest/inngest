import { useEffect, useRef, useState, type ReactNode } from 'react';
import {
  CardElement,
  Elements,
  useElements,
  useStripe,
} from '@stripe/react-stripe-js';
import { loadStripe } from '@stripe/stripe-js';
import { Input } from '@inngest/components/Forms/Input';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';
import { placeholderColor, textColor } from '@/utils/tailwind';
import { cardErrors } from './copy';
import type { CardFieldsProps } from './types';
import type { useComputeVerification } from './useComputeVerification';

const key = import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY;
const stripePromise = key ? loadStripe(key) : null;
type Controller = ReturnType<typeof useComputeVerification>;
type Adapter = {
  cardReady: boolean;
  onSubmitCard: () => Promise<void>;
  renderCardFields: (props: CardFieldsProps) => ReactNode;
};

export function StripeVerification({
  controller,
  children,
}: {
  controller: Controller;
  children: (adapter: Adapter) => ReactNode;
}) {
  return (
    <Elements stripe={stripePromise}>
      <StripeFields controller={controller}>{children}</StripeFields>
    </Elements>
  );
}
function StripeFields({
  controller,
  children,
}: {
  controller: Controller;
  children: (adapter: Adapter) => ReactNode;
}) {
  const stripe = useStripe();
  const elements = useElements();
  const [name, setName] = useState('');
  const [zip, setZip] = useState('');
  const [complete, setComplete] = useState(false);
  const [error, setError] = useState<string>();
  const [fieldError, setFieldError] = useState<string>();
  const [pending, setPending] = useState(false);
  const intent = useRef<string>();
  const secret = useRef<string>();
  const busy = useRef(false);
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  async function submit() {
    if (!stripe || !elements || busy.current) return;
    busy.current = true;
    setPending(true);
    setError(undefined);
    try {
      if (!intent.current && !secret.current) {
        const card = elements.getElement(CardElement);
        if (!card || !name.trim() || !/^\d{5}(?:-\d{4})?$/.test(zip)) return;
        const result = await stripe.createPaymentMethod({
          type: 'card',
          card,
          billing_details: {
            name: name.trim(),
            address: { postal_code: zip, country: 'US' },
          },
        });
        if (!mounted.current) return;
        if (result.error) {
          setError(
            result.error.message ?? 'Could not read this card. Try again.',
          );
          return;
        }
        const added = await controller.run({
          kind: 'card',
          paymentMethodID: result.paymentMethod.id,
        });
        if (!mounted.current || !added || added.error) return;
        secret.current = added.cardClientSecret ?? undefined;
      }
      if (secret.current) {
        const result = await stripe.handleNextAction({
          clientSecret: secret.current,
        });
        if (!mounted.current) return;
        if (result.error) {
          if (result.error.type === 'api_connection_error') {
            // The SetupIntent is still waiting for 3D Secure; retry it.
            setError(cardErrors.temporary);
            return;
          }
          // The user closed the popup or the bank refused 3D Secure, so this
          // SetupIntent can't succeed. The next submit starts over with the card.
          secret.current = undefined;
          setError(
            'Card authentication didn’t complete. Try again or use a different card.',
          );
          return;
        }
        if (!result.setupIntent) {
          setError('Could not finish card authentication. Try again.');
          return;
        }
        intent.current = result.setupIntent.id;
        secret.current = undefined;
      }
      if (intent.current) {
        const finished = await controller.run({
          kind: 'finish-card',
          setupIntentID: intent.current,
        });
        if (finished && !finished.error) {
          secret.current = finished.cardClientSecret ?? undefined;
          intent.current = undefined;
        }
      }
    } catch {
      if (mounted.current) setError('Could not verify this card. Try again.');
    } finally {
      busy.current = false;
      if (mounted.current) setPending(false);
    }
  }

  return children({
    cardReady: Boolean(
      stripe &&
        (intent.current ||
          secret.current ||
          (complete && name.trim() && /^\d{5}(?:-\d{4})?$/.test(zip))),
    ),
    onSubmitCard: submit,
    renderCardFields: ({ disabled, errorID }) => (
      <div className="flex flex-col gap-4" aria-busy={pending}>
        <label className="text-basis flex flex-col gap-1 text-sm">
          Name on card
          <Input
            autoComplete="cc-name"
            value={name}
            disabled={disabled || pending}
            onChange={(event) => {
              intent.current = undefined;
              secret.current = undefined;
              setName(event.target.value);
            }}
            aria-invalid={controller.transient.failure === 'CARD_NAME_REQUIRED'}
            aria-describedby={
              controller.transient.failure === 'CARD_NAME_REQUIRED'
                ? errorID
                : undefined
            }
          />
        </label>
        <div
          role="group"
          aria-labelledby="verification-card-label"
          aria-describedby={fieldError ? 'stripe-verification-error' : errorID}
          className="flex flex-col gap-1"
        >
          <p id="verification-card-label" className="text-basis text-sm">
            Card details
          </p>
          <div
            id="verification-stripe-card"
            className="border-muted bg-canvasBase rounded border p-3"
          >
            <CardElement
              options={{
                disableLink: true,
                hidePostalCode: true,
                disabled: disabled || pending,
                style: {
                  base: {
                    fontSize: '14px',
                    color: resolveColor(textColor.basis, isDark()),
                    '::placeholder': {
                      color: resolveColor(placeholderColor.disabled, isDark()),
                    },
                  },
                  invalid: { color: resolveColor(textColor.error, isDark()) },
                },
              }}
              onChange={(event) => {
                if (!busy.current && !event.empty) {
                  intent.current = undefined;
                  secret.current = undefined;
                }
                setComplete(event.complete);
                setFieldError(event.error?.message);
                setError(undefined);
              }}
            />
          </div>
        </div>
        <label className="text-basis flex flex-col gap-1 text-sm">
          ZIP code
          <Input
            autoComplete="billing postal-code"
            inputMode="numeric"
            value={zip}
            disabled={disabled || pending}
            onChange={(event) => {
              intent.current = undefined;
              secret.current = undefined;
              setZip(event.target.value);
            }}
          />
        </label>
        {(error || fieldError || !stripePromise) && (
          <p
            role="alert"
            id="stripe-verification-error"
            className="text-error text-sm"
          >
            {error ??
              fieldError ??
              'Card verification is unavailable. Try again later.'}
          </p>
        )}
      </div>
    ),
  });
}
