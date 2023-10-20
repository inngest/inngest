'use client';

import { ArrowPathIcon, ExclamationCircleIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';

type VercelIntegrationErrorProps = {
  error: Error;
  reset: () => void;
};

export default function VercelIntegrationError({ reset }: VercelIntegrationErrorProps) {
  return (
    <div className="flex h-full w-full flex-col items-center justify-center gap-5">
      <div className="inline-flex items-center gap-2 text-red-600">
        <ExclamationCircleIcon className="h-4 w-4" />
        <h2 className="text-sm">Failed to load Vercel integration settings</h2>
      </div>
      <Button
        appearance="outlined"
        iconSide="right"
        icon={<ArrowPathIcon className=" text-slate-700" />}
        btnAction={() => reset()}
        label="Reload"
      />
    </div>
  );
}
