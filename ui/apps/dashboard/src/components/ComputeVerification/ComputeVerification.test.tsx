// @vitest-environment jsdom
import React from 'react';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ComputeVerification } from './ComputeVerification';
import {
  initialState,
  personal,
  scenarioState,
  scenarios,
  type ScenarioID,
} from './demo/scenarios';
import { VerificationDemo } from './demo/VerificationDemo';
import { MockCardFields } from './demo/MockCardFields';
import { getVerificationScreen } from './state';
import type { VerificationActions, VerificationState } from './types';

vi.mock('@/utils/useGraphQLQuery', async () => import('./demo/shellServices'));
vi.mock('@/queries', async () => import('./demo/shellServices'));
vi.mock(
  '../NavigationV2/QuickSearch/data',
  async () => import('./demo/shellServices'),
);
vi.mock(
  '../Onboarding/useOnboardingStep',
  async () => import('./demo/shellServices'),
);
vi.mock(
  '@clerk/tanstack-react-start',
  async () => import('./demo/shellServices'),
);

beforeEach(() => {
  window.scrollTo = vi.fn();
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: vi.fn(() => ({
      matches: false,
      addListener: vi.fn(),
      removeListener: vi.fn(),
    })),
  });
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  window.history.replaceState(null, '', '/');
});
const callbacks = (): VerificationActions => ({
  onSendCode: vi.fn(),
  onVerifyCode: vi.fn(),
  onResendCode: vi.fn(),
  onChangeNumber: vi.fn(),
  onSubmitCard: vi.fn(),
  onContinueEmail: vi.fn(),
  onManageEmail: vi.fn(),
  onSwitchCompany: vi.fn(),
  onSwitchProfile: vi.fn(),
  onRetry: vi.fn(),
  onReturn: vi.fn(),
  onContactSupport: vi.fn(),
});
function setup(state: Partial<VerificationState> = {}, actions = callbacks()) {
  return {
    actions,
    ...render(
      React.createElement(ComputeVerification, {
        state: { ...initialState, ...state },
        actions,
      }),
    ),
  };
}

describe('supplied verification state', () => {
  it('starts at the first incomplete required check and keeps blocked states distinct', () => {
    expect(getVerificationScreen(initialState)).toBe('phone');
    expect(getVerificationScreen(scenarioState('saved'))).toBe('card');
    expect(getVerificationScreen(scenarioState('admin'))).toBe(
      'admin-required',
    );
    expect(getVerificationScreen(scenarioState('email'))).toBe('email');
    expect(getVerificationScreen(scenarioState('sms-attempts'))).toBe('code');
    expect(getVerificationScreen(scenarioState('locked'))).toBe('locked');
    expect(getVerificationScreen(scenarioState('suspended'))).toBe('suspended');
    expect(getVerificationScreen(scenarioState('complete-cli'))).toBe(
      'complete',
    );
  });
  it('keeps mismatched company or profile in front of forms and completion', () => {
    for (const id of ['mismatch', 'profile-mismatch'] as const)
      expect(getVerificationScreen(scenarioState(id))).toBe('mismatch');
    expect(
      getVerificationScreen({
        ...scenarioState('complete-cli'),
        selectedAccount: personal,
      }),
    ).toBe('mismatch');
  });
  it.each(Object.keys(scenarios) as ScenarioID[])(
    'renders the %s review scenario',
    (id) => {
      setup(scenarioState(id));
      expect(
        screen.queryByRole('heading') || screen.queryByRole('status'),
      ).not.toBeNull();
    },
  );
  it('shows the admin-required explanation without a status breakdown or active forms', () => {
    setup(scenarioState('admin'));
    expect(screen.getByRole('heading').textContent).toBe(
      'Ask an admin to verify',
    );
    expect(screen.queryByText('Verified')).toBeNull();
    expect(screen.queryByText('Incomplete')).toBeNull();
    expect(screen.queryByRole('textbox')).toBeNull();
  });
});

describe('phone and code interactions', () => {
  it('validates missing/malformed numbers, formats a pasted +1 number, and preserves input after a failure', async () => {
    const { actions, rerender } = setup();
    fireEvent.click(screen.getByRole('button', { name: 'Send code' }));
    expect(screen.getByRole('alert').textContent).toBe(
      'Enter your mobile number.',
    );
    const input = screen.getByLabelText<HTMLInputElement>('Mobile number');
    fireEvent.change(input, { target: { value: '123' } });
    fireEvent.click(screen.getByRole('button', { name: 'Send code' }));
    expect(actions.onSendCode).not.toHaveBeenCalled();
    fireEvent.change(input, { target: { value: '+1 (415) 555-0132' } });
    expect(input.value).toBe('(415) 555-0132');
    fireEvent.click(screen.getByRole('button', { name: 'Send code' }));
    await waitFor(() =>
      expect(actions.onSendCode).toHaveBeenCalledWith('+14155550132'),
    );
    rerender(
      <ComputeVerification
        state={{ ...initialState, phoneError: 'temporary' }}
        actions={actions}
      />,
    );
    expect(input.value).toBe('(415) 555-0132');
    expect(screen.getByRole('alert').id).toBe(
      input.getAttribute('aria-describedby'),
    );
  });
  it('blocks rapid duplicate submission before supplied pending updates', async () => {
    let finish!: () => void;
    const actions = callbacks();
    actions.onSendCode = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          finish = resolve;
        }),
    );
    setup({}, actions);
    fireEvent.change(screen.getByLabelText('Mobile number'), {
      target: { value: '4155550132' },
    });
    const form = screen.getByLabelText('Mobile number').closest('form')!;
    fireEvent.submit(form);
    fireEvent.submit(form);
    expect(actions.onSendCode).toHaveBeenCalledTimes(1);
    expect(
      screen.getByRole<HTMLButtonElement>('button', { name: 'Sending code' })
        .disabled,
    ).toBe(true);
    await act(async () => finish());
  });
  it('supports full OTP paste, arrows, backspace and autofill, and sends exactly six digits', async () => {
    const { actions } = setup(scenarioState('sms'));
    const first = screen.getByLabelText<HTMLInputElement>('Digit 1');
    const last = screen.getByLabelText<HTMLInputElement>('Digit 6');
    const verify = screen.getByRole<HTMLButtonElement>('button', {
      name: 'Verify phone',
    });
    expect(verify.disabled).toBe(true);
    expect(first.autocomplete).toBe('one-time-code');
    fireEvent.paste(first, { clipboardData: { getData: () => '123 456' } });
    expect(last.value).toBe('6');
    expect(document.activeElement).toBe(last);
    expect(verify.disabled).toBe(false);
    fireEvent.keyDown(last, { key: 'ArrowLeft' });
    expect(document.activeElement).toBe(screen.getByLabelText('Digit 5'));
    fireEvent.keyDown(last, { key: 'Backspace' });
    expect(last.value).toBe('');
    expect(verify.disabled).toBe(true);
    fireEvent.keyDown(last, { key: 'Backspace' });
    expect(screen.getByLabelText<HTMLInputElement>('Digit 5').value).toBe('');
    fireEvent.change(first, { target: { value: '123456' } });
    fireEvent.click(verify);
    await waitFor(() =>
      expect(actions.onVerifyCode).toHaveBeenCalledWith('123456'),
    );
  });
  it('keeps cooldown and attempt-limit actions disabled, but lets expired codes be resent', async () => {
    const actions = callbacks();
    const { rerender } = setup(scenarioState('sms-cooldown'), actions);
    expect(
      screen.getByRole<HTMLButtonElement>('button', { name: 'Resend in 0:30' })
        .disabled,
    ).toBe(true);
    rerender(
      <ComputeVerification
        state={scenarioState('sms-expired')}
        actions={actions}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Request new code' }));
    await waitFor(() => expect(actions.onResendCode).toHaveBeenCalledOnce());
    rerender(
      <ComputeVerification
        state={scenarioState('sms-attempts')}
        actions={actions}
      />,
    );
    expect(screen.getByLabelText<HTMLInputElement>('Digit 1').disabled).toBe(
      true,
    );
    expect(
      screen.getByRole<HTMLButtonElement>('button', { name: 'Verify phone' })
        .disabled,
    ).toBe(true);
    expect(
      screen.getByRole<HTMLButtonElement>('button', { name: 'Change number' })
        .disabled,
    ).toBe(true);
  });
  it('clears local inputs and errors when account context changes', () => {
    const { actions, rerender } = setup();
    fireEvent.change(screen.getByLabelText('Mobile number'), {
      target: { value: '123' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Send code' }));
    rerender(
      <ComputeVerification
        state={{
          ...initialState,
          selectedAccount: personal,
          originatingAccount: personal,
        }}
        actions={actions}
      />,
    );
    expect(screen.getByLabelText<HTMLInputElement>('Mobile number').value).toBe(
      '',
    );
    expect(screen.queryByRole('alert')).toBeNull();
  });
});

describe('card, email, and completion', () => {
  it('shows general card failures without marking individual fields invalid', () => {
    render(
      <ComputeVerification
        state={scenarioState('card-temporary')}
        actions={callbacks()}
        cardReady
        renderCardFields={(props) => <MockCardFields {...props} populated />}
      />,
    );
    expect(screen.getByRole('alert').textContent).toBe(
      'We couldn’t verify this card. Try again.',
    );
    expect(
      screen
        .getByRole('group', { name: 'Card details' })
        .getAttribute('aria-describedby'),
    ).toBe('card-error');
    for (const input of screen.getAllByRole('textbox')) {
      expect(input.getAttribute('aria-invalid')).not.toBe('true');
    }
  });
  it('collects no card data without a supplied field adapter', () => {
    setup(scenarioState('saved'));
    expect(screen.queryByRole('textbox')).toBeNull();
    expect(
      screen.getByRole<HTMLButtonElement>('button', { name: 'Verify card' })
        .disabled,
    ).toBe(true);
  });
  it('exposes email callbacks without advancing supplied state itself', async () => {
    const { actions } = setup(scenarioState('email'));
    fireEvent.click(
      screen.getByRole('button', { name: 'Manage email in your profile' }),
    );
    await waitFor(() => expect(actions.onManageEmail).toHaveBeenCalledOnce());
    fireEvent.click(
      screen.getByRole('button', { name: 'Continue verification' }),
    );
    await waitFor(() => expect(actions.onContinueEmail).toHaveBeenCalledOnce());
    expect(screen.getByRole('heading').textContent).toBe('Verify your email');
  });
  it.each([
    ['complete-cli', null],
    ['complete-dashboard', 'Back to dashboard'],
    ['complete-api', null],
  ] as const)('uses entry-specific completion for %s', (scenario, label) => {
    const { actions } = setup(scenarioState(scenario));
    if (label) {
      fireEvent.click(screen.getByRole('button', { name: label }));
      expect(actions.onReturn).toHaveBeenCalledOnce();
    } else {
      expect(
        screen.getByText(
          scenario === 'complete-cli'
            ? 'Return to your terminal to continue your deploy.'
            : 'You can now use Compute and Sandboxes. Retry your original request.',
        ),
      ).toBeTruthy();
      expect(screen.queryByRole('button')).toBeNull();
      expect(actions.onReturn).not.toHaveBeenCalled();
    }
  });
  it('runs the entire mock flow, clears code on change-number, and ends at completion', async () => {
    vi.useFakeTimers();
    render(<VerificationDemo />);
    await act(async () => vi.advanceTimersByTimeAsync(0));
    fireEvent.change(screen.getByLabelText('Mobile number'), {
      target: { value: '4155550132' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Send code' }));
    await act(async () => vi.advanceTimersByTimeAsync(650));
    fireEvent.change(screen.getByLabelText('Digit 1'), {
      target: { value: '111111' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Change number' }));
    expect(screen.getByLabelText<HTMLInputElement>('Mobile number').value).toBe(
      '(415) 555-0132',
    );
    fireEvent.click(screen.getByRole('button', { name: 'Send code' }));
    await act(async () => vi.advanceTimersByTimeAsync(650));
    expect(screen.getByLabelText<HTMLInputElement>('Digit 1').value).toBe('');
    fireEvent.change(screen.getByLabelText('Digit 1'), {
      target: { value: '123456' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Verify phone' }));
    await act(async () => vi.advanceTimersByTimeAsync(650));
    expect(screen.getByRole('heading').textContent).toBe('Verify a card');
    expect(
      screen.getByRole<HTMLButtonElement>('button', { name: 'Verify card' })
        .disabled,
    ).toBe(false);
    expect(screen.getByLabelText<HTMLInputElement>('Card number').value).toBe(
      '4242 4242 4242 4242',
    );
    expect(
      screen.getByLabelText<HTMLInputElement>('Card number').readOnly,
    ).toBe(true);
    fireEvent.click(screen.getByRole('button', { name: 'Verify card' }));
    await act(async () => vi.advanceTimersByTimeAsync(650));
    expect(screen.getByRole('heading').textContent).toBe(
      'Verification completed!',
    );
  });
  it('populates saved-progress card fixtures from the review controls', async () => {
    window.history.replaceState(null, '', '/?scenario=saved');
    render(<VerificationDemo />);
    await screen.findByRole('heading', { name: 'Verify a card' });
    expect(
      screen.getByRole<HTMLButtonElement>('button', { name: 'Verify card' })
        .disabled,
    ).toBe(true);
    fireEvent.click(screen.getByText('Development demo · scenarios'));
    fireEvent.click(screen.getByLabelText('Populate dummy card'));
    expect(screen.getByLabelText<HTMLInputElement>('Card number').value).toBe(
      '4242 4242 4242 4242',
    );
    expect(
      screen.getByRole<HTMLButtonElement>('button', { name: 'Verify card' })
        .disabled,
    ).toBe(false);
  });
  it('uses the existing company menu and waits for an explicit choice from multiple organizations', async () => {
    window.history.replaceState(null, '', '/?scenario=mismatch');
    render(<VerificationDemo />);
    const trigger = await screen.findByRole('button', {
      name: 'P Personal Workspace',
    });
    fireEvent.keyDown(trigger, { key: 'Enter' });
    fireEvent.click(
      await screen.findByRole('menuitem', { name: 'Switch organisations' }),
    );
    await screen.findByRole('heading', { name: 'Choose a company' });
    expect(
      screen.getByRole('button', { name: 'P Personal Workspace' }),
    ).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Northstar Labs' })).toBeTruthy();
    expect(
      screen.getByRole('button', { name: 'Personal Workspace' }),
    ).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Acme Research' }));
    await screen.findByRole('button', { name: 'A Acme Research' });
    expect(screen.getByRole('heading').textContent).toBe(
      'Check your company and profile',
    );
    expect(
      screen.queryByRole('button', { name: 'company switcher' }),
    ).toBeNull();
  });
});
