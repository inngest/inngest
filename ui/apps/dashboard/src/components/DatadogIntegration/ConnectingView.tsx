'use client';

import { Alert } from '@inngest/components/Alert';
import { IconSpinner } from '@inngest/components/icons/Spinner';

type Props = {
  errorMessage?: string;
};

export default function SetupPage({ errorMessage }: Props) {
  if (errorMessage) {
    // TODO(cdzombak): "An API key with this name already exists" -> needs user action to resolve

    return (
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
    );
  }

  return (
    <div className="flex flex-row gap-4 pl-3.5">
      <IconSpinner className="fill-link h-8 w-8" />
      <div className="text-lg">Please wait while we connect you to Datadog…</div>
    </div>
  );
}
