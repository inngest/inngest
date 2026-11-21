import { useEffect, useRef, useState, type ReactNode } from 'react';
import { Button } from '@inngest/components/Button';
import type { ButtonProps } from '@inngest/components/Button/Button';
import { cn } from '@inngest/components/utils/classNames';
import phone from './assets/phone.svg';
import card from './assets/card.svg';
import email from './assets/email.svg';
import success from './assets/success.svg';
import calendar from './assets/calendar.svg';
import lock from './assets/lock.svg';
import timer from './assets/timer.svg';
import suspended from './assets/suspended.svg';

const icons = { phone, card, email, success, calendar, lock, timer, suspended };
export function VerificationIcon({
  kind,
  large = false,
}: {
  kind: keyof typeof icons;
  large?: boolean;
}) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        'flex shrink-0 items-center justify-center',
        kind === 'success'
          ? 'bg-primary-3xSubtle h-11 w-11 rounded-full'
          : large || ['calendar', 'lock', 'timer'].includes(kind)
          ? 'bg-canvasSubtle h-11 w-11 rounded-md'
          : 'bg-canvasSubtle h-[34px] w-[34px] rounded-md',
      )}
    >
      <img
        alt=""
        src={icons[kind]}
        className={kind === 'success' ? undefined : 'dark:invert'}
      />
    </span>
  );
}

export function Panel({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <section
      className={cn(
        'border-subtle bg-canvasBase flex w-full flex-col gap-5 rounded-lg border p-6',
        className,
      )}
    >
      {children}
    </section>
  );
}

export function Heading({
  title,
  step,
  children,
  small = false,
}: {
  title: string;
  step?: 1 | 2;
  children?: ReactNode;
  small?: boolean;
}) {
  return (
    <div className="flex flex-col gap-1">
      {step && <p className="text-muted text-xs">Step {step} of 2</p>}
      <h1
        tabIndex={-1}
        className={cn(
          'text-basis font-medium outline-none',
          small ? 'text-xl leading-7' : 'text-2xl leading-8',
        )}
      >
        {title}
      </h1>
      {children && (
        <div className="text-subtle text-sm leading-5">{children}</div>
      )}
    </div>
  );
}

export function ErrorMessage({
  id,
  children,
}: {
  id: string;
  children?: ReactNode;
}) {
  return children ? (
    <p id={id} role="alert" className="text-error text-sm leading-5">
      {children}
    </p>
  ) : null;
}

export function ActionButton(props: ButtonProps) {
  return <Button size="large" className="w-full text-sm" {...props} />;
}

/** Keep rapid clicks and rejected adapters safe even before pending props are updated. */
export function useAction(suppliedPending = false) {
  const busy = useRef(false);
  const mounted = useRef(true);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string>();
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  async function run(action: () => void | Promise<void>) {
    if (busy.current || suppliedPending) return;
    busy.current = true;
    setPending(true);
    setError(undefined);
    try {
      await action();
    } catch {
      if (mounted.current) setError('Something went wrong. Try again.');
    } finally {
      busy.current = false;
      if (mounted.current) setPending(false);
    }
  }
  return { pending: pending || suppliedPending, error, run };
}
