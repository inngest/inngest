import { Button } from '@inngest/components/Button';
import OrgAvatar from '../Layout/OrgAvatar';
import { completionCopy } from './copy';
import {
  ActionButton,
  ErrorMessage,
  Heading,
  Panel,
  useAction,
  VerificationIcon,
} from './shared';
import type { VerificationActions, VerificationState } from './types';

type Props = { state: VerificationState; actions: VerificationActions };

export function EmailScreen({ state, actions }: Props) {
  const action = useAction(state.pending);
  return (
    <Panel>
      <VerificationIcon kind="email" />
      <Heading title="Verify your email">
        Verify your primary email in your profile before continuing. If you use
        a temporary email address, switch to another address.
      </Heading>
      <Button
        appearance="ghost"
        className="self-start px-0 text-sm"
        label="Manage email in your profile"
        disabled={action.pending}
        onClick={() => void action.run(actions.onManageEmail)}
      />
      <p className="text-muted text-xs">
        Once your email is verified, return here to continue.
      </p>
      {state.emailError && (
        <span role="status" className="sr-only">
          {state.emailError}
        </span>
      )}
      <ErrorMessage id="email-error">{action.error}</ErrorMessage>
      <ActionButton
        label="Continue verification"
        loading={action.pending}
        disabled={action.pending}
        onClick={() => void action.run(actions.onContinueEmail)}
      />
      {action.pending && (
        <span role="status" className="sr-only">
          Checking email…
        </span>
      )}
    </Panel>
  );
}

export function MismatchScreen({ state }: Props) {
  const origin = state.originatingAccount!;
  return (
    <Panel>
      <Heading title="Check your company and profile">
        This {state.entrySource === 'cli' ? 'deploy' : 'request'} is for{' '}
        {origin.companyName}. Select {origin.companyName} in the company
        switcher to continue. If you’re signed in with the wrong profile, switch
        it from the profile menu.
      </Heading>
      <div className="border-subtle flex flex-col gap-[18px] rounded border px-4 py-2.5">
        {[
          {
            label:
              state.entrySource === 'cli' ? 'In terminal' : 'Original request',
            account: origin,
          },
          { label: 'In browser', account: state.selectedAccount },
        ].map(({ label, account }) => (
          <div key={label} className="flex flex-col gap-1.5">
            <p className="text-muted text-[11px] leading-4">{label}</p>
            <div className="flex items-center gap-2">
              <OrgAvatar
                size="lg"
                profile={{
                  orgName: account.companyName,
                  displayName: account.displayName,
                  orgProfilePic: null,
                  userProfilePic: null,
                  isMarketplace: false,
                  marketplace: null,
                }}
              />
              <div className="flex min-w-0 flex-col gap-0.5">
                <p className="text-basis break-words text-sm">
                  {account.companyName}
                </p>
                <p className="text-muted break-all text-xs">
                  {account.maskedEmail}
                </p>
              </div>
            </div>
          </div>
        ))}
      </div>
    </Panel>
  );
}

export function CompletionScreen({ state, actions }: Props) {
  const copy = completionCopy[state.entrySource];
  return (
    <div className="flex flex-col items-center gap-6 pt-8 text-center">
      <VerificationIcon kind="success" />
      <Heading title="Verification completed!">{copy.description}</Heading>
      {copy.label && (
        <ActionButton
          className="text-sm"
          label={copy.label}
          onClick={actions.onReturn}
        />
      )}
    </div>
  );
}

const blockedCopy = {
  'early-access-unavailable': {
    icon: 'calendar',
    title: 'Early access isn’t available yet',
    description: 'This account doesn’t have early access to Compute.',
    label: 'Back to dashboard',
  },
  locked: {
    icon: 'lock',
    title: 'Too many verification attempts',
    description:
      'Verification is locked for this account. Contact support if you think this is a mistake.',
    label: 'Contact support',
  },
  unavailable: {
    icon: 'timer',
    title: 'Verification is temporarily unavailable',
    description:
      'We couldn’t complete the check. Your completed checks are saved. Try again later.',
    label: 'Try again',
  },
  suspended: {
    icon: 'suspended',
    title: 'Compute and Sandboxes are suspended',
    description:
      'This account’s access has been suspended. Contact support if you think this is a mistake.',
    label: 'Contact support',
  },
  'admin-required': {
    icon: 'lock',
    title: 'Ask an admin to verify',
    description: 'An account admin needs to complete verification.',
    label: 'Back to dashboard',
  },
} as const;

export function BlockedScreen({
  state,
  actions,
  screen,
}: Props & { screen: keyof typeof blockedCopy }) {
  const copy = blockedCopy[screen];
  const action = useAction(state.pending);
  const handler =
    screen === 'early-access-unavailable'
      ? actions.onReturn
      : screen === 'unavailable'
      ? actions.onRetry
      : screen === 'admin-required'
      ? actions.onReturn
      : actions.onContactSupport;
  return (
    <Panel>
      <VerificationIcon kind={copy.icon} large />
      <Heading small title={copy.title}>
        {copy.description}
      </Heading>
      <ErrorMessage id="blocked-error">{action.error}</ErrorMessage>
      <ActionButton
        label={copy.label}
        loading={action.pending}
        disabled={action.pending}
        onClick={() => void action.run(handler)}
      />
    </Panel>
  );
}
