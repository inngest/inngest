'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@clerk/nextjs';
import { Button } from '@inngest/components/Button';
import { Error as ErrorElement } from '@inngest/components/Error/Error';
import * as Sentry from '@sentry/nextjs';

type UserSetupErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function UserSetupError({ error }: UserSetupErrorProps) {
  const { signOut } = useAuth();
  const router = useRouter();

  useEffect(() => {
    Sentry.captureException(new Error('Failed to set up user', { cause: error }));
  }, [error]);

  return (
    <ErrorElement
      message="Failed to set up your user"
      button={
        <Button
          label="Contact Support"
          appearance="outlined"
          onClick={() => {
            signOut(() => router.push('/support'));
          }}
        />
      }
    />
  );
}
