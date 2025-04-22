'use client';

import { Alert } from '@inngest/components/Alert';
import { IconSpinner } from '@inngest/components/icons/Spinner';

type Props = {
  errorMessage?: string;
};

export default function SetupPage({ errorMessage }: Props) {
  if (errorMessage) {
    if (errorMessage.toLowerCase().includes('api key with this name already exists')) {
      return (
        <Alert severity="warning" className="text-base">
          This Datadog organization was previously connected to Inngest, and you’ll need to remove
          Inngest’s old API key from your Datadog account manually before reconnecting.
          <Alert.Link severity="warning" target="_blank" href="https://www.inngest.com/docs/">
            Please see our documentation for instructions.
          </Alert.Link>
          {/* TODO(cdzombak): update docs link once it's written */}
        </Alert>
      );
    }

    return (
      <Alert severity="error">
        Connection failed.
        <Alert.Link severity="error" target="_blank" href="/support">
          Please contact Inngest support if this error persists.
        </Alert.Link>{' '}
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
