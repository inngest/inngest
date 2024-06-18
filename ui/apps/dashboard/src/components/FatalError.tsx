'use client';

import { useEffect } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import * as Sentry from '@sentry/nextjs';

import { pathCreator } from '@/utils/urls';

type Props = {
  error: Error & { digest?: string };
  reset: () => void;
};

export function FatalError({ error, reset }: Props) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <div className="mt-32 flex w-full flex-col items-center justify-center gap-4">
      <Alert severity="error">
        <p className="mb-4 font-semibold">{error.message}</p>

        <p>
          An error occurred! Refresh the page to try again. If the problem persists, contact
          support.
        </p>
      </Alert>

      <div className="flex gap-4">
        <Button btnAction={() => reset()} kind="primary" label="Try again" />

        <Link href={pathCreator.support()}>Contact support</Link>
      </div>
    </div>
  );
}
