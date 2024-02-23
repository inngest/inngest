'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@clerk/nextjs';
import { ExclamationCircleIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
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
    <div className="flex h-full w-full flex-col items-center justify-center gap-5">
      <div className="inline-flex items-center gap-2 text-red-600">
        <ExclamationCircleIcon className="h-4 w-4" />
        <h2 className="text-sm">Failed to set up your user</h2>
      </div>
      <Button
        label="Contact Support"
        appearance="outlined"
        btnAction={() => {
          signOut(() => router.push('/support'));
        }}
      />
    </div>
  );
}
