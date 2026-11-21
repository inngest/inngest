import type { CardError, CodeError, EntrySource, PhoneError } from './types';

export const phoneErrors: Record<PhoneError, string> = {
  unsupported: 'Use a US mobile number. VoIP numbers are unsupported.',
  'already-used':
    'This phone number is already used to verify another Inngest account. Use a different US mobile number.',
  temporary: 'We couldn’t send the code. Try again.',
};
export const codeErrors: Record<CodeError, string> = {
  incorrect: 'That code isn’t correct. Check the code and try again.',
  expired: 'This code has expired. Request a new code.',
  'too-many-attempts': 'Too many attempts. Try again later.',
  temporary: 'We couldn’t verify the code. Try again.',
};
export const cardErrors: Record<CardError, string> = {
  'non-us': 'Use a card issued in the United States to continue.',
  prepaid:
    'Prepaid cards are unsupported. Use a US-issued credit or debit card.',
  'already-used':
    'This card is already used to verify another Inngest account. Use a different card.',
  bank: 'Your bank couldn’t verify this card. Try a different US-issued credit or debit card.',
  temporary: 'We couldn’t verify this card. Try again.',
};
export const completionCopy: Record<
  EntrySource,
  { description: string; label: string | null }
> = {
  cli: {
    description: 'Return to your terminal to continue your deploy.',
    label: null,
  },
  dashboard: {
    description: 'You can now use Compute and Sandboxes for this account.',
    label: 'Back to dashboard',
  },
  api: {
    description:
      'You can now use Compute and Sandboxes. Retry your original request.',
    label: null,
  },
};
