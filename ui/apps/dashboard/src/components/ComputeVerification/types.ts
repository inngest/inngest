import type { ReactNode } from 'react';

export type EntrySource = 'cli' | 'dashboard' | 'api';
export type AccountContext = {
  accountID: string;
  companyName: string;
  profileID: string;
  displayName: string;
  maskedEmail: string;
};

export type PhoneError = 'unsupported' | 'already-used' | 'temporary';
export type CodeError =
  | 'incorrect'
  | 'expired'
  | 'too-many-attempts'
  | 'temporary';
export type CardError =
  | 'name-required'
  | 'non-us'
  | 'prepaid'
  | 'already-used'
  | 'bank'
  | 'temporary';
export type VerificationState = {
  selectedAccount: AccountContext;
  originatingAccount?: AccountContext;
  entrySource: EntrySource;
  canVerify: boolean;
  status:
    | 'complete'
    | 'rate-limited'
    | 'finalizing'
    | 'loading'
    | 'verification'
    | 'early-access-unavailable'
    | 'locked'
    | 'suspended'
    | 'unavailable';
  email: 'complete' | 'needs-attention';
  phone: 'incomplete' | 'code-sent' | 'complete';
  card: 'incomplete' | 'complete';
  /** Only masked verified details belong in supplied state. */
  verifiedPhone?: string;
  verifiedCard?: string;
  destinationPhone?: string;
  phoneError?: PhoneError;
  codeError?: CodeError;
  cardError?: CardError;
  emailError?: string;
  pending?: boolean;
  /** Remaining seconds are supplied by the owner; the UI does not enforce a limit. */
  resendCooldownSeconds?: number;
};

type Action = () => void | Promise<void>;
export type VerificationActions = {
  onSendCode: (phone: string) => void | Promise<void>;
  onVerifyCode: (code: string) => void | Promise<void>;
  onResendCode: Action;
  onChangeNumber: Action;
  /** The future Stripe adapter owns card details; this callback receives no card data. */
  onSubmitCard: Action;
  onContinueEmail: Action;
  onManageEmail: Action;
  onSwitchCompany: Action;
  onSwitchProfile: Action;
  onRetry: Action;
  onReturn: Action;
  onContactSupport: Action;
};

export type CardFieldsProps = {
  disabled: boolean;
  /** Form-level verification message. Only field-specific validation should mark an input invalid. */
  errorID?: string;
};
export type VerificationProps = {
  state: VerificationState;
  actions: VerificationActions;
  /** Replace with vendor-controlled fields at integration time. Never pass raw card data. */
  renderCardFields?: (props: CardFieldsProps) => ReactNode;
  cardReady?: boolean;
};
