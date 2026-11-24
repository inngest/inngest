import type { VerificationState } from './types';

export function getVerificationScreen(state: VerificationState) {
  if (state.status === 'loading') return 'loading';
  const origin = state.originatingAccount;
  if (
    origin &&
    (origin.accountID !== state.selectedAccount.accountID ||
      origin.profileID !== state.selectedAccount.profileID)
  )
    return 'mismatch';
  if (state.status !== 'verification') return state.status;
  if (
    state.email === 'complete' &&
    state.phone === 'complete' &&
    state.card === 'complete'
  ) {
    return 'complete';
  }
  if (!state.canVerify) return 'admin-required';
  if (state.email === 'needs-attention') return 'email';
  if (state.phone === 'code-sent') return 'code';
  if (state.phone !== 'complete') return 'phone';
  return 'card';
}

export function normalizePhone(value: string) {
  const digits = value.replace(/\D/g, '');
  return digits.length === 11 && digits.startsWith('1')
    ? digits.slice(1)
    : digits;
}

export function isValidPhone(value: string) {
  return /^[2-9]\d{2}[2-9]\d{6}$/.test(normalizePhone(value));
}

export function formatPhone(value: string) {
  const digits = normalizePhone(value);
  if (digits.length > 10) return value;
  if (digits.length <= 3) return digits;
  if (digits.length <= 6) return `(${digits.slice(0, 3)}) ${digits.slice(3)}`;
  return `(${digits.slice(0, 3)}) ${digits.slice(3, 6)}-${digits.slice(6)}`;
}
