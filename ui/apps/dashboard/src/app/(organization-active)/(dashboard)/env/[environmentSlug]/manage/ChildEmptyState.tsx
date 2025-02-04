'use client';

import type { Route } from 'next';
import { Button } from '@inngest/components/Button';

import { staticSlugs } from '@/utils/environments';

export default function ChildEmptyState() {
  return (
    <div className="h-full w-full overflow-y-scroll py-16">
      <div className="mx-auto flex w-[640px] flex-col gap-4">
        <div className="border-subtle rounded-md border px-8 pt-8">
          <h3 className="flex items-center text-xl font-semibold">
            Manage Keys for All Branch Environments
          </h3>
          <p className="text-subtle mt-2 text-sm font-medium normal-case">
            Keys are shared for all branch environments. The Inngest SDK can automatically route
            your events to the correct branch.
          </p>
          <div className="border-subtle mt-6 flex items-center gap-2 border-t py-4">
            <Button
              kind="primary"
              appearance="outlined"
              href={`/env/${staticSlugs.branch}/manage/keys` as Route}
              label="Manage Event Keys"
            />

            <Button
              kind="primary"
              appearance="outlined"
              href={`/env/${staticSlugs.branch}/manage/signing-key` as Route}
              label="Manage Signing Keys"
            />
          </div>
        </div>
      </div>
    </div>
  );
}
