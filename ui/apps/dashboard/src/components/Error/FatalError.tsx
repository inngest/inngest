import { useEffect } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import * as Sentry from '@sentry/tanstackstart-react';

import { pathCreator } from '@/utils/urls';
import { useRouter } from '@tanstack/react-router';

type Props = {
  error: Error & { digest?: string };
};

export function FatalError({ error }: Props) {
  const router = useRouter();
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <div className="m-auto mt-32 flex w-fit flex-col gap-4">
      <Alert
        severity="error"
        button={
          <Button
            onClick={() => router.invalidate()}
            kind="secondary"
            appearance="outlined"
            label="Refresh page"
          />
        }
      >
        <p className="mb-4 font-semibold">{error.message}</p>

        <p>
          An error occurred! Refresh the page to try again. If the problem
          persists, contact{' '}
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
