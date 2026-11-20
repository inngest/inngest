import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { codeErrors, phoneErrors } from './copy';
import { formatPhone, isValidPhone, normalizePhone } from './state';
import {
  ActionButton,
  ErrorMessage,
  Heading,
  Panel,
  useAction,
} from './shared';
import { VerificationSteps } from './VerificationSteps';
import { OTPInput } from './OTPInput';
import type { VerificationActions, VerificationState } from './types';

type Props = { state: VerificationState; actions: VerificationActions };

export function PhoneScreen({
  state,
  actions,
  phone,
  setPhone,
}: Props & {
  phone: string;
  setPhone: (phone: string) => void;
}) {
  const action = useAction(state.pending);
  const [validationError, setValidationError] = useState<string>();
  const [edited, setEdited] = useState(false);
  const error =
    validationError ??
    action.error ??
    (!edited && state.phoneError ? phoneErrors[state.phoneError] : undefined);
  return (
    <Panel>
      <VerificationSteps phone={state.phone} card={state.card} />
      <Heading step={1} title="Verify your phone">
        Use a US mobile number. VoIP numbers are unsupported.
      </Heading>
      <form
        noValidate
        className="flex flex-col gap-5"
        aria-busy={action.pending}
        onSubmit={(event) => {
          event.preventDefault();
          if (action.pending) return;
          if (!isValidPhone(phone)) {
            setValidationError(
              phone.trim()
                ? 'Enter a valid 10-digit US phone number.'
                : 'Enter your mobile number.',
            );
            event.currentTarget.querySelector('input')?.focus();
            return;
          }
          setValidationError(undefined);
          setEdited(false);
          void action.run(() =>
            actions.onSendCode(`+1${normalizePhone(phone)}`),
          );
        }}
      >
        <div className="flex flex-col gap-1">
          <label htmlFor="verification-phone" className="text-basis text-sm">
            Mobile number
          </label>
          <div className="grid grid-cols-[78px_minmax(0,1fr)] gap-2">
            <div
              aria-label="Country calling code +1"
              className="border-muted text-disabled flex h-10 items-center rounded border px-3 text-sm"
            >
              +1
            </div>
            <Input
              id="verification-phone"
              type="tel"
              inputMode="tel"
              autoComplete="tel-national"
              placeholder="(415) 555-0132"
              aria-invalid={Boolean(error)}
              aria-describedby={error ? 'phone-error' : undefined}
              className="h-10"
              disabled={action.pending}
              value={phone}
              onChange={(event) => {
                setPhone(formatPhone(event.target.value));
                setValidationError(undefined);
                setEdited(true);
              }}
            />
          </div>
          <ErrorMessage id="phone-error">{error}</ErrorMessage>
        </div>
        <ActionButton
          type="submit"
          label="Send code"
          loading={action.pending}
          disabled={action.pending}
          aria-label={action.pending ? 'Sending code' : 'Send code'}
        />
        {action.pending && (
          <span className="sr-only" role="status">
            Sending code…
          </span>
        )}
      </form>
    </Panel>
  );
}

export function CodeScreen({ state, actions }: Props) {
  const [code, setCode] = useState<string[]>(Array(6).fill(''));
  const [edited, setEdited] = useState(false);
  const action = useAction(state.pending);
  const limited = state.codeError === 'too-many-attempts';
  const expired = state.codeError === 'expired';
  const error =
    action.error ??
    (state.codeError && (!edited || limited || expired)
      ? codeErrors[state.codeError]
      : undefined);
  const cooldown = Math.max(0, Math.ceil(state.resendCooldownSeconds ?? 0));
  const destination = state.destinationPhone ?? 'your mobile number';
  return (
    <Panel>
      <VerificationSteps phone={state.phone} card={state.card} />
      <Heading step={1} title="Enter your verification code">
        <div className="flex flex-wrap items-center gap-x-0.5 gap-y-1">
          <p>
            Enter the code sent to{' '}
            <span className="font-medium">{destination}.</span>
          </p>
          <Button
            size="small"
            appearance="ghost"
            label="Change number"
            disabled={action.pending || limited}
            onClick={() => void action.run(actions.onChangeNumber)}
          />
        </div>
      </Heading>
      <form
        className="flex flex-col gap-5"
        aria-busy={action.pending}
        onSubmit={(event) => {
          event.preventDefault();
          if (
            code.some((digit) => !digit) ||
            action.pending ||
            limited ||
            expired
          )
            return;
          setEdited(false);
          void action.run(() => actions.onVerifyCode(code.join('')));
        }}
      >
        <div className="flex flex-col gap-1">
          <OTPInput
            value={code}
            onChange={(value) => {
              setCode(value);
              setEdited(true);
            }}
            disabled={action.pending || limited}
            errorID={error ? 'code-error' : undefined}
          />
          <ErrorMessage id="code-error">{error}</ErrorMessage>
          <p
            className={`text-muted text-xs ${
              cooldown === 0 ? 'invisible' : ''
            }`}
            aria-hidden={cooldown === 0}
          >
            You can request another code when the timer ends.
          </p>
        </div>
        <div className="flex flex-col gap-2">
          <ActionButton
            type="submit"
            label={
              state.codeError === 'temporary' ? 'Try again' : 'Verify phone'
            }
            loading={action.pending}
            disabled={
              action.pending ||
              limited ||
              expired ||
              code.some((digit) => !digit)
            }
            aria-label={action.pending ? 'Verifying phone' : undefined}
          />
          <Button
            appearance="ghost"
            className="w-full text-sm tabular-nums"
            size="large"
            label={
              cooldown > 0
                ? `Resend in ${Math.floor(cooldown / 60)}:${String(
                    cooldown % 60,
                  ).padStart(2, '0')}`
                : expired
                ? 'Request new code'
                : 'Resend code'
            }
            disabled={action.pending || limited || cooldown > 0}
            onClick={() => {
              void action.run(async () => {
                await actions.onResendCode();
                setCode(Array(6).fill(''));
                setEdited(false);
              });
            }}
          />
        </div>
        <span role="status" className="sr-only">
          {action.pending
            ? 'Checking verification…'
            : cooldown === 0
            ? 'You can request a new code.'
            : ''}
        </span>
      </form>
    </Panel>
  );
}
