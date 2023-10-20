'use client';

import { usePathname } from 'next/navigation';
import { ExclamationCircleIcon } from '@heroicons/react/20/solid';

import { getManageKey } from '@/utils/urls';

export default function EventKeyError() {
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
      <div className="inline-flex items-center gap-2 text-red-600">
        <ExclamationCircleIcon className="h-4 w-4" />
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
