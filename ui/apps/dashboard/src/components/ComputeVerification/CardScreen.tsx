import { cardErrors } from './copy';
import {
  ActionButton,
  ErrorMessage,
  Heading,
  Panel,
  useAction,
  VerificationIcon,
} from './shared';
import type { VerificationProps } from './types';

export function CardScreen({
  state,
  actions,
  renderCardFields,
  cardReady = false,
}: VerificationProps) {
  const action = useAction(state.pending);
  const error =
    action.error ?? (state.cardError ? cardErrors[state.cardError] : undefined);
  return (
    <Panel>
      <VerificationIcon kind="card" />
      <Heading step={2} title="Verify a card">
        Use a US credit or debit card for verification. It becomes your default
        payment method but can be changed later.
      </Heading>
      <form
        className="flex flex-col gap-5"
        aria-busy={action.pending}
        onSubmit={(event) => {
          event.preventDefault();
          if (action.pending || !cardReady || !renderCardFields) return;
          void action.run(actions.onSubmitCard);
        }}
      >
        {renderCardFields ? (
          renderCardFields({
            disabled: action.pending,
            errorID: error ? 'card-error' : undefined,
          })
        ) : (
          <div
            className="border-subtle bg-canvasSubtle text-muted rounded border p-4 text-sm"
            role="status"
          >
            Card verification is unavailable. Try again later.
          </div>
        )}
        <ErrorMessage id="card-error">{error}</ErrorMessage>
        <div className="border-subtle bg-canvasSubtle flex items-center justify-between gap-3 rounded-md border p-3">
          <div>
            <p className="text-basis text-sm">You won’t be charged</p>
            <p className="text-muted mt-0.5 text-xs">
              Card verification via Stripe
            </p>
          </div>
          <p className="text-basis text-xl">$0.00</p>
        </div>
        <ActionButton
          type="submit"
          label={action.pending ? 'Verifying card…' : 'Verify card'}
          loading={action.pending}
          disabled={action.pending || !cardReady || !renderCardFields}
          aria-label={action.pending ? 'Verifying card' : 'Verify card'}
        />
        {action.pending && (
          <span role="status" className="sr-only">
            Verifying card…
          </span>
        )}
      </form>
    </Panel>
  );
}
