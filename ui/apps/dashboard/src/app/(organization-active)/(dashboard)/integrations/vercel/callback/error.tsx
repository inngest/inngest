'use client';

import { useEffect } from 'react';
import { RiErrorWarningLine } from '@remixicon/react';
import * as Sentry from '@sentry/nextjs';

type VercelIntegrationCallbackErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function VercelIntegrationCallbackError({
  error,
}: VercelIntegrationCallbackErrorProps) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <div className="flex h-full w-full flex-col items-center justify-center gap-5">
      <div className="inline-flex items-center gap-2 text-yellow-600">
        <RiErrorWarningLine className="h-4 w-4" />
        <h2 className="text-sm">
          Failed to set up Inngest integration. Please close the window and try to add the
          integration again.
        </h2>
      </div>
    </div>
  );
}
