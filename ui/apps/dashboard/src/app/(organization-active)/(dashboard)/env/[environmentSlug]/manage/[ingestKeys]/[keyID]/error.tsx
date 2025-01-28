'use client';

import { useEffect } from 'react';
import { usePathname } from 'next/navigation';
import { RiErrorWarningLine } from '@remixicon/react';
import * as Sentry from '@sentry/nextjs';

import { getManageKey } from '@/utils/urls';

type EventKeyErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function EventKeyError({ error }: EventKeyErrorProps) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  const pathname = usePathname();
  const page = getManageKey(pathname);

  let label = '';
  switch (page) {
    case 'keys':
      label = 'Event Key';
      break;
    case 'webhooks':
      label = 'Webhook';
      break;
  }

  return (
    <div className="flex h-full w-full flex-col items-center justify-center gap-5">
      <div className="text-error inline-flex items-center gap-2">
        <RiErrorWarningLine className="h-4 w-4" />
        <h2 className="text-sm">
          {'Failed to load this ' +
            label +
            '. This ' +
            label +
            ' will likely not belong to this environment.'}
        </h2>
      </div>
    </div>
  );
}
