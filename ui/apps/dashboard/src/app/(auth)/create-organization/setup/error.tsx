'use client';

import { useEffect } from 'react';
import { ExclamationCircleIcon } from '@heroicons/react/20/solid';
import ArrowPathIcon from '@heroicons/react/20/solid/ArrowPathIcon';
import { Button } from '@inngest/components/Button';
import * as Sentry from '@sentry/nextjs';

type OrganizationSetupErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function OrganizationSetupError({ error, reset }: OrganizationSetupErrorProps) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <div className="flex h-full w-full flex-col items-center justify-center gap-5">
      <div className="inline-flex items-center gap-2 text-red-600">
        <ExclamationCircleIcon className="h-4 w-4" />
        <h2 className="text-sm">Failed to set up your organization</h2>
      </div>
      <Button
        appearance="outlined"
        iconSide="right"
        icon={<ArrowPathIcon className=" text-slate-700" />}
        btnAction={() => reset()}
        label="Try Again"
      />
    </div>
  );
}
