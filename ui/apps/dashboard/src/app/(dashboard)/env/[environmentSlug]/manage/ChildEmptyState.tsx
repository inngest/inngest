'use client';

import type { Route } from 'next';
import { ExclamationTriangleIcon } from '@heroicons/react/20/solid';

import Button from '@/components/Button';
import { staticSlugs } from '@/utils/environments';

export default function ChildEmptyState() {
  return (
    <div className="h-full w-full overflow-y-scroll py-16">
      <div className="mx-auto flex w-[640px] flex-col gap-4">
        <div className="rounded-lg border border-slate-300 px-8 pt-8">
          <h3 className="flex items-center text-xl font-semibold text-slate-800">
            Manage Keys for All Branch Environments
          </h3>
          <p className="mt-2 text-sm font-medium normal-case text-slate-500">
            Keys are shared for all branch environments. The Inngest SDK can automatically route
            your events to the correct branch.
          </p>
          <div className="mt-6 flex items-center gap-2 border-t border-slate-100 py-4">
            <Button variant="primary" href={`/env/${staticSlugs.branch}/manage/keys` as Route}>
              Manage
            </Button>
            {/* <Button className="ml-auto" variant="secondary" target="_blank" href={'' as Route}>
              Learn More About Branch Environments
            </Button> */}
          </div>
        </div>
      </div>
    </div>
  );
}
