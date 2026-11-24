import { useId } from 'react';
import { Button } from '@inngest/components/Button';
import { Pill } from '@inngest/components/Pill';
import ProgressBar from '@inngest/components/ProgressBar/ProgressBar';
import { Skeleton } from '@inngest/components/Skeleton';

// Supplied by the parent. An access entitlement alone cannot determine this state.
export type SandboxesVerificationState =
  | { status: 'blocked' }
  | { status: 'loading' }
  | { status: 'error' }
  | { status: 'verified' | 'not-required' }
  | { status: 'required'; canVerify: false }
  | {
      status: 'required';
      canVerify: true;
      started: boolean;
      progress: { completed: number; total: number };
    };

export type SandboxesVerificationCalloutProps = {
  state: SandboxesVerificationState;
  /** The host owns navigation, account context, and resuming the flow. */
  onVerify: () => void;
  onRetry: () => void;
};

export function showsVerificationCallout(state: SandboxesVerificationState) {
  return state.status !== 'verified' && state.status !== 'not-required';
}

export function SandboxesVerificationCallout({
  state,
  onVerify,
  onRetry,
}: SandboxesVerificationCalloutProps) {
  const headingID = useId();

  if (!showsVerificationCallout(state)) return null;

  return (
    <section
      aria-labelledby={headingID}
      className="border-subtle bg-canvasBase flex flex-col items-start gap-4 rounded-lg border px-4 py-3 shadow-sm"
    >
      {state.status === 'blocked' ? (
        <>
          <h2 id={headingID} className="text-basis text-base font-medium">
            Compute and Sandboxes are suspended
          </h2>
          <p className="text-muted text-sm">
            Contact support to review this account’s access.
          </p>
        </>
      ) : state.status === 'loading' ? (
        <div role="status" aria-busy="true" className="w-full">
          <h2 id={headingID} className="sr-only">
            Account verification
          </h2>
          <span className="sr-only">Loading verification status…</span>
          <div aria-hidden="true" className="flex flex-col gap-4">
            <Skeleton className="h-6 w-40 motion-reduce:before:animate-none" />
            <div className="flex flex-col gap-2">
              <Skeleton className="h-3 w-full motion-reduce:before:animate-none" />
              <Skeleton className="h-3 w-full motion-reduce:before:animate-none" />
              <Skeleton className="h-3 w-3/4 motion-reduce:before:animate-none" />
            </div>
            <Skeleton className="h-1 w-full motion-reduce:before:animate-none" />
            <Skeleton className="h-8 w-36 motion-reduce:before:animate-none" />
          </div>
        </div>
      ) : state.status === 'error' ? (
        <>
          <div role="alert">
            <h2 id={headingID} className="text-basis text-base font-medium">
              Unable to load verification
            </h2>
            <p className="text-muted mt-1 text-sm">
              We couldn’t check your verification status. Try again.
            </p>
          </div>
          <Button
            kind="secondary"
            appearance="outlined"
            label="Try again"
            onClick={onRetry}
          />
        </>
      ) : state.status === 'required' ? (
        state.canVerify ? (
          <>
            <div>
              <h2 id={headingID} className="text-basis text-base font-medium">
                Verify your account
              </h2>
              <p className="text-muted mt-1 text-sm">
                To help prevent abuse, we verify your phone and a US-issued
                credit or debit card. You won’t be charged for verification.
              </p>
            </div>
            <VerificationProgress {...state.progress} />
            <Button
              kind="primary"
              appearance={state.started ? 'outlined' : 'solid'}
              label={
                state.started ? 'Continue verification' : 'Start verification'
              }
              onClick={onVerify}
            />
          </>
        ) : (
          <>
            <div>
              <h2 id={headingID} className="text-basis text-base font-medium">
                Ask an admin to verify
              </h2>
              <p className="text-muted mt-1 text-sm">
                An account admin needs to complete verification.
              </p>
            </div>
            <Pill kind="default" appearance="solid">
              Unverified
            </Pill>
          </>
        )
      ) : null}
    </section>
  );
}

function VerificationProgress({
  completed,
  total,
}: {
  completed: number;
  total: number;
}) {
  if (!Number.isInteger(total) || total <= 0 || !Number.isFinite(completed))
    return null;

  const value = Math.max(0, Math.min(Math.trunc(completed), total));

  return (
    <div
      role="progressbar"
      aria-label="Account verification"
      aria-valuemin={0}
      aria-valuemax={total}
      aria-valuenow={value}
      aria-valuetext={`${value} of ${total} checks complete`}
      className="flex w-full gap-1"
    >
      {Array.from({ length: total }, (_, index) => (
        <div key={index} aria-hidden="true" className="min-w-0 flex-1">
          <ProgressBar
            size="small"
            limit={1}
            value={index < value ? 1 : 0}
            className="bg-canvasMuted"
          />
        </div>
      ))}
    </div>
  );
}
