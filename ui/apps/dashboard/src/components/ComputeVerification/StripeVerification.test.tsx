// @vitest-environment jsdom
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { StripeVerification } from './StripeVerification';
import type { useComputeVerification } from './useComputeVerification';
const mocks = vi.hoisted(() => ({
  createPaymentMethod: vi.fn(),
  handleNextAction: vi.fn(),
  run: vi.fn(),
  options: vi.fn(),
  getElement: vi.fn(() => ({})),
}));
vi.mock('@stripe/stripe-js', () => ({
  loadStripe: vi.fn(() => Promise.resolve({})),
}));
vi.mock('@stripe/react-stripe-js', () => ({
  Elements: ({ children }: { children: React.ReactNode }) => children,
  useStripe: () => mocks,
  useElements: () => ({ getElement: mocks.getElement }),
  CardElement: ({
    options,
    onChange,
  }: {
    options: unknown;
    onChange: (event: unknown) => void;
  }) => {
    mocks.options(options);
    return (
      <button type="button" onClick={() => onChange({ complete: true })}>
        Complete hosted card fields
      </button>
    );
  },
}));
beforeEach(() => {
  mocks.createPaymentMethod.mockResolvedValue({
    paymentMethod: { id: 'pm_test' },
  });
});
afterEach(() => {
  cleanup();
  vi.resetAllMocks();
});
function setup() {
  const controller = { run: mocks.run, transient: {} } as unknown as ReturnType<
    typeof useComputeVerification
  >;
  render(
    <StripeVerification controller={controller}>
      {(adapter) => (
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void adapter.onSubmitCard();
          }}
        >
          {adapter.renderCardFields({ disabled: false })}
          <button type="submit" disabled={!adapter.cardReady}>
            Submit card
          </button>
        </form>
      )}
    </StripeVerification>,
  );
  fireEvent.change(screen.getByLabelText('Name on card'), {
    target: { value: 'Test User' },
  });
  fireEvent.change(screen.getByLabelText('ZIP code'), {
    target: { value: '94107' },
  });
  fireEvent.click(
    screen.getByRole('button', { name: 'Complete hosted card fields' }),
  );
}
describe('Stripe verification orchestration', () => {
  it('creates a PaymentMethod with name and ZIP, disables Link, and never creates a token', async () => {
    mocks.run.mockResolvedValue({
      error: null,
      verification: null,
      cardClientSecret: null,
    });
    setup();
    fireEvent.click(screen.getByRole('button', { name: 'Submit card' }));
    await waitFor(() =>
      expect(mocks.run).toHaveBeenCalledWith({
        kind: 'card',
        paymentMethodID: 'pm_test',
      }),
    );
    expect(mocks.createPaymentMethod.mock.calls[0][0]).toMatchObject({
      type: 'card',
      billing_details: {
        name: 'Test User',
        address: { postal_code: '94107', country: 'US' },
      },
    });
    expect(mocks.options.mock.calls[0][0]).toMatchObject({
      disableLink: true,
      hidePostalCode: true,
    });
    expect(mocks.handleNextAction).not.toHaveBeenCalled();
  });
  it('handles 3DS and retries only the finish mutation after an uncertain finish', async () => {
    mocks.run
      .mockResolvedValueOnce({
        error: null,
        cardClientSecret: 'seti_secret',
        verification: null,
      })
      .mockResolvedValueOnce(undefined)
      .mockResolvedValueOnce({ error: null, cardClientSecret: null });
    mocks.handleNextAction.mockResolvedValue({
      setupIntent: { id: 'seti_test', status: 'succeeded' },
    });
    setup();
    fireEvent.click(screen.getByRole('button', { name: 'Submit card' }));
    await waitFor(() => expect(mocks.run).toHaveBeenCalledTimes(2));
    expect(mocks.handleNextAction).toHaveBeenCalledWith({
      clientSecret: 'seti_secret',
    });
    expect(mocks.run.mock.calls[1][0]).toEqual({
      kind: 'finish-card',
      setupIntentID: 'seti_test',
    });
    fireEvent.click(screen.getByRole('button', { name: 'Submit card' }));
    await waitFor(() => expect(mocks.run).toHaveBeenCalledTimes(3));
    expect(mocks.createPaymentMethod).toHaveBeenCalledTimes(1);
    expect(mocks.handleNextAction).toHaveBeenCalledTimes(1);
    expect(mocks.run.mock.calls[2][0]).toEqual({
      kind: 'finish-card',
      setupIntentID: 'seti_test',
    });
  });
  it('stops after canceled authentication without claiming completion', async () => {
    mocks.run.mockResolvedValue({
      error: null,
      cardClientSecret: 'seti_secret',
    });
    mocks.handleNextAction.mockResolvedValue({
      error: { message: 'Authentication canceled' },
    });
    setup();
    fireEvent.click(screen.getByRole('button', { name: 'Submit card' }));
    await screen.findByText(/Your bank couldn’t verify this card/);
    expect(mocks.run).toHaveBeenCalledTimes(1);
    // The failed SetupIntent is dropped, so the next submit starts with a new card.
    fireEvent.click(screen.getByRole('button', { name: 'Submit card' }));
    await waitFor(() =>
      expect(mocks.createPaymentMethod).toHaveBeenCalledTimes(2),
    );
  });
  it('never submits a payment method after Stripe validation fails', async () => {
    mocks.createPaymentMethod.mockResolvedValue({
      error: { message: 'Invalid card' },
    });
    setup();
    fireEvent.click(screen.getByRole('button', { name: 'Submit card' }));
    await screen.findByText('Invalid card');
    expect(mocks.run).not.toHaveBeenCalled();
  });
});
