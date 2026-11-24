import type {
  ComputeVerificationQuery,
  ComputeVerificationError,
} from '@/gql/graphql';
import type { AccountContext, EntrySource, VerificationState } from './types';

export type Summary = NonNullable<
  ComputeVerificationQuery['account']['computeVerification']
>;
export type VerificationFailure = ComputeVerificationError | 'NETWORK';
export type TransientVerification = {
  phoneNumber?: string;
  cooldown?: number;
  failure?: VerificationFailure;
  action?: 'email' | 'send' | 'code' | 'card';
};

export function adaptVerification(
  summary: Summary | null | undefined,
  selectedAccount: AccountContext,
  targetAccountID: string,
  source: EntrySource,
  transient: TransientVerification = {},
): VerificationState {
  const passed = (kind: string) =>
    summary?.checks.find((check) => check.kind === kind)?.passed === true;
  const state: VerificationState = {
    selectedAccount,
    originatingAccount:
      targetAccountID === selectedAccount.accountID
        ? undefined
        : {
            ...selectedAccount,
            accountID: targetAccountID,
            companyName: targetAccountID,
            maskedEmail: '',
          },
    entrySource: source,
    canVerify: summary?.canVerify ?? false,
    status: 'unavailable',
    email: passed('EMAIL') ? 'complete' : 'needs-attention',
    phone: passed('PHONE')
      ? 'complete'
      : transient.phoneNumber
      ? 'code-sent'
      : 'incomplete',
    card: passed('CARD') ? 'complete' : 'incomplete',
    destinationPhone: transient.phoneNumber,
    resendCooldownSeconds: transient.cooldown ?? 0,
    verifiedPhone:
      summary?.checks.find((check) => check.kind === 'PHONE')?.display ??
      undefined,
    verifiedCard:
      summary?.checks.find((check) => check.kind === 'CARD')?.display ??
      undefined,
  };
  if (!summary) return state;
  switch (summary.status) {
    case 'VERIFIED':
    case 'EXEMPT':
      state.status = 'complete';
      return state;
    case 'SUSPENDED':
      state.status = 'suspended';
      return state;
    case 'NOT_ELIGIBLE':
      state.status = 'early-access-unavailable';
      return state;
    case 'REQUIRED':
      state.status =
        passed('EMAIL') && passed('PHONE') && passed('CARD')
          ? 'finalizing'
          : 'verification';
      break;
    default:
      return state;
  }
  if (transient.failure === 'NOT_ADMIN') state.canVerify = false;
  // Status and permission decisions take precedence over per-attempt errors.
  if (!state.canVerify) {
    state.status = 'verification';
    return state;
  }
  switch (transient.failure) {
    case 'RATE_LIMITED':
      state.status = 'rate-limited';
      break;
    case 'NOT_ELIGIBLE':
      state.status = 'unavailable';
      break;
    case 'EMAIL_UNVERIFIED':
    case 'EMAIL_DISPOSABLE':
      state.email = 'needs-attention';
      state.status = 'verification';
      break;
    case 'PHONE_INVALID':
    case 'PHONE_NOT_US':
    case 'PHONE_NOT_MOBILE':
      state.phone = 'incomplete';
      state.phoneError = 'unsupported';
      break;
    case 'PHONE_IN_USE':
      state.phone = 'incomplete';
      state.phoneError = 'already-used';
      break;
    case 'CODE_INCORRECT':
      state.codeError = 'incorrect';
      break;
    case 'CODE_EXPIRED':
      state.codeError = 'expired';
      break;
    case 'CARD_NAME_REQUIRED':
      state.cardError = 'name-required';
      break;
    case 'CARD_NOT_US':
      state.cardError = 'non-us';
      break;
    case 'CARD_PREPAID':
      state.cardError = 'prepaid';
      break;
    case 'CARD_DECLINED':
      state.cardError = 'bank';
      break;
    case 'CARD_IN_USE':
      state.cardError = 'already-used';
      break;
    case 'UNAVAILABLE':
    case 'NETWORK':
      state.status = 'unavailable';
      break;
  }
  // The email check runs on its own when the page opens. Show the email screen
  // only once a check came back, not while the first one is in flight.
  if (
    state.status === 'verification' &&
    state.email === 'needs-attention' &&
    !state.originatingAccount &&
    transient.action !== 'email'
  )
    state.status = 'loading';
  return state;
}

/** UI list/detail gate only. The API still enforces operation permissions and entitlements. */
export function allowsSandboxList(
  status: Summary['status'] | undefined,
  enabled: boolean,
) {
  // Without a status, such as when the verification read fails, keep the plan's entitlement.
  if (!status) return enabled;
  if (status === 'REQUIRED' || status === 'SUSPENDED') return false;
  return status === 'VERIFIED' || status === 'EXEMPT' || enabled;
}
