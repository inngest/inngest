import { useEffect, useRef, useState, type ReactNode } from 'react';
import { VerificationSkeleton } from './VerificationSkeleton';
import { CardScreen } from './CardScreen';
import {
  BlockedScreen,
  CompletionScreen,
  EmailScreen,
  MismatchScreen,
} from './ContextScreens';
import { CodeScreen, PhoneScreen } from './PhoneScreens';
import { getVerificationScreen } from './state';
import type { VerificationProps } from './types';
import './verification.css';

/** Supply the existing application header and footer, with no sidebar. */
export function VerificationLayout({
  header,
  footer,
  children,
}: {
  header: ReactNode;
  footer: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="bg-canvasSubtle text-basis flex min-h-dvh flex-col font-sans">
      {header}
      <main className="border-subtle bg-canvasBase mx-2 flex flex-1 flex-col items-center rounded-md border px-4 pb-8 pt-8 sm:pt-16">
        {children}
      </main>
      {footer}
    </div>
  );
}

export function ComputeVerification(props: VerificationProps) {
  // Account/profile changes discard every local input and error. Supplied progress
  // belongs to the caller; no state is copied between accounts here.
  const account = props.state.selectedAccount;
  return (
    <VerificationContent
      key={`${account.accountID}:${account.profileID}`}
      {...props}
    />
  );
}

function VerificationContent(props: VerificationProps) {
  const { state, actions } = props;
  const screen = getVerificationScreen(state);
  const [phone, setPhone] = useState('');
  const container = useRef<HTMLDivElement>(null);
  useEffect(() => {
    container.current?.querySelector<HTMLElement>('h1')?.focus();
  }, [screen]);
  let content: ReactNode;
  switch (screen) {
    case 'loading':
      content = <VerificationSkeleton />;
      break;
    case 'mismatch':
      content = <MismatchScreen state={state} actions={actions} />;
      break;
    case 'email':
      content = <EmailScreen state={state} actions={actions} />;
      break;
    case 'phone':
      content = (
        <PhoneScreen
          state={state}
          actions={actions}
          phone={phone}
          setPhone={setPhone}
        />
      );
      break;
    case 'code':
      content = <CodeScreen state={state} actions={actions} />;
      break;
    case 'card':
      content = <CardScreen {...props} />;
      break;
    case 'complete':
      content = <CompletionScreen state={state} actions={actions} />;
      break;
    default:
      content = (
        <BlockedScreen state={state} actions={actions} screen={screen} />
      );
  }
  return (
    <div
      ref={container}
      className={`compute-verification w-full ${
        screen === 'unavailable' ? 'max-w-[482px]' : 'max-w-md'
      }`}
    >
      {content}
    </div>
  );
}
