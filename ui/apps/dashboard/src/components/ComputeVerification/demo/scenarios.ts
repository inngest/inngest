import type { AccountContext, VerificationState } from '../types';

export const company: AccountContext = {
  accountID: 'northstar',
  companyName: 'Northstar Labs',
  profileID: 'sanjana',
  displayName: 'Sanjana',
  maskedEmail: 's•••••@northstar.example',
};
export const personal: AccountContext = {
  accountID: 'personal',
  companyName: 'Personal Workspace',
  profileID: 'personal-profile',
  displayName: 'Sanjana',
  maskedEmail: 's•••••@personal.example',
};
export const initialState: VerificationState = {
  selectedAccount: company,
  originatingAccount: company,
  entrySource: 'cli',
  canVerify: true,
  status: 'verification',
  email: 'complete',
  phone: 'incomplete',
  card: 'incomplete',
};
type Scenario = {
  label: string;
  state: Partial<VerificationState>;
  populatedCard?: boolean;
};
const code = {
  phone: 'code-sent',
  destinationPhone: '+1 (415) 555-0132',
} as const;
const card = { phone: 'complete', verifiedPhone: '+1 ••• ••• 0132' } as const;
const complete = {
  ...card,
  card: 'complete',
  verifiedCard: 'Visa •••• 4242',
} as const;
export const scenarios = {
  happy: { label: 'Full happy path', state: {} },
  loading: { label: 'Initial loading', state: { status: 'loading' } },
  'phone-pending': { label: 'Phone · sending', state: { pending: true } },
  'phone-unsupported': {
    label: 'Phone · unsupported',
    state: { phoneError: 'unsupported' },
  },
  'phone-used': {
    label: 'Phone · already used',
    state: { phoneError: 'already-used' },
  },
  'phone-temporary': {
    label: 'Phone · temporary failure',
    state: { phoneError: 'temporary' },
  },
  sms: { label: 'SMS · default', state: code },
  'sms-pending': {
    label: 'SMS · submitting',
    state: { ...code, pending: true },
  },
  'sms-incorrect': {
    label: 'SMS · incorrect',
    state: { ...code, codeError: 'incorrect' },
  },
  'sms-expired': {
    label: 'SMS · expired',
    state: { ...code, codeError: 'expired' },
  },
  'sms-cooldown': {
    label: 'SMS · resend cooldown',
    state: { ...code, resendCooldownSeconds: 30 },
  },
  'sms-attempts': {
    label: 'SMS · too many attempts',
    state: { ...code, codeError: 'too-many-attempts' },
  },
  'sms-temporary': {
    label: 'SMS · temporary failure',
    state: { ...code, codeError: 'temporary' },
  },
  saved: { label: 'Saved progress · card directly', state: card },
  'card-populated': {
    label: 'Card · populated dummy fixture',
    state: card,
    populatedCard: true,
  },
  'card-pending': {
    label: 'Card · loading',
    state: { ...card, pending: true },
    populatedCard: true,
  },
  'card-non-us': {
    label: 'Card · non-US',
    state: { ...card, cardError: 'non-us' },
    populatedCard: true,
  },
  'card-prepaid': {
    label: 'Card · prepaid',
    state: { ...card, cardError: 'prepaid' },
    populatedCard: true,
  },
  'card-used': {
    label: 'Card · already used',
    state: { ...card, cardError: 'already-used' },
    populatedCard: true,
  },
  'card-bank': {
    label: 'Card · bank failure',
    state: { ...card, cardError: 'bank' },
    populatedCard: true,
  },
  'card-temporary': {
    label: 'Card · temporary failure',
    state: { ...card, cardError: 'temporary' },
    populatedCard: true,
  },
  email: { label: 'Email · recovery', state: { email: 'needs-attention' } },
  'email-pending': {
    label: 'Email · checking',
    state: { email: 'needs-attention', pending: true },
  },
  'email-attention': {
    label: 'Email · still needs attention',
    state: {
      email: 'needs-attention',
      emailError:
        'Your primary email still needs attention. Verify or replace it in your profile.',
    },
  },
  mismatch: {
    label: 'Account and profile mismatch',
    state: { selectedAccount: personal },
  },
  'profile-mismatch': {
    label: 'Profile mismatch only',
    state: {
      selectedAccount: {
        ...company,
        profileID: 'another-profile',
        maskedEmail: 's•••••@personal.example',
      },
    },
  },
  admin: {
    label: 'Admin required · partial progress',
    state: { ...card, canVerify: false },
  },
  unavailable: {
    label: 'Temporarily unavailable',
    state: { ...card, status: 'unavailable' },
  },
  locked: { label: 'Verification locked', state: { status: 'locked' } },
  suspended: {
    label: 'Access suspended',
    state: { ...complete, status: 'suspended' },
  },
  'early-unavailable': {
    label: 'Early access unavailable',
    state: { status: 'early-access-unavailable' },
  },
  'complete-cli': { label: 'Completion · CLI', state: complete },
  'complete-dashboard': {
    label: 'Completion · dashboard',
    state: { ...complete, entrySource: 'dashboard' },
  },
  'complete-api': {
    label: 'Completion · API/direct',
    state: { ...complete, entrySource: 'api' },
  },
} satisfies Record<string, Scenario>;
export type ScenarioID = keyof typeof scenarios;
export function scenarioState(id: ScenarioID): VerificationState {
  return { ...initialState, ...scenarios[id].state };
}
