'use client';

import { Alert } from '@inngest/components/Alert';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { IconDatadog } from '@inngest/components/icons/platforms/Datadog';

type Props = {
  errorMessage?: string;
};

export default function SetupPage({ errorMessage }: Props) {
  return (
    <div className="mx-auto mt-16 flex w-[800px] flex-col">
      <div className="text-basis mb-14 flex flex-row items-center justify-start text-2xl font-medium">
        <div className="bg-contrast mr-4 flex h-12 w-12 items-center justify-center rounded">
          <IconDatadog className="text-onContrast" size={20} />
        </div>
        Connect to Datadog
      </div>
      {errorMessage ? (
        <>
          <Alert severity="error">
            Connection failed. Please{' '}
            <a href="/support" className="underline">
              contact Inngest support
            </a>{' '}
            if this error persists.
            <br />
            <br />
            <code>{errorMessage}</code>
          </Alert>
        </>
      ) : (
        <>
          <div className="flex flex-row gap-4 pl-3.5">
            <IconSpinner className="fill-link h-8 w-8" />
            <div className="text-lg">Please wait while we connect you to Datadog…</div>
          </div>
        </>
      )}
    </div>
  );
}
