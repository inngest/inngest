'use client';

import { useEffect } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
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
    <div className="m-auto mt-32 flex w-fit flex-col gap-4">
      <Alert
        severity="error"
        button={
          <Button
            onClick={() => reset()}
            kind="secondary"
            appearance="outlined"
            label="Refresh page"
          />
        }
      >
        <p className="mb-4 font-semibold">{error.message}</p>

        <p>
          An error occurred! Refresh the page to try again. If the problem persists, contact{' '}
          <Alert.Link
            size="medium"
            severity="error"
            className="inline-flex"
            href={pathCreator.support()}
          >
            Inngest support
          </Alert.Link>
          .
        </p>
      </Alert>
    </div>
  );
}
