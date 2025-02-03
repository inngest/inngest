'use client';

import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button/Button';
import { Card } from '@inngest/components/Card/Card';
import { Time } from '@inngest/components/Time';
import { RiLinkUnlinkM } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';

type Props = {
  className?: string;
  envSlug: string;
  latestSyncTime: Date;
};

export function UnattachedSyncsCard({ envSlug, latestSyncTime }: Props) {
  const router = useRouter();
  return (
    <Card className="mb-6">
      <div className="text-basis p-6">
        <div className="items-top flex justify-between">
          <div className="flex items-center gap-3">
            <div className="bg-canvasSubtle text-subtle border-subtle rounded-md border p-3">
              <RiLinkUnlinkM className="h-7 w-7" />
            </div>
            <div>
              <div className="mb-0.5 flex items-center gap-2 text-xl">Unattached syncs</div>
              <p className="text-muted text-sm">
                Unattached syncs are failed syncs that could not be associated with an app.
              </p>
            </div>
          </div>

          <Button
            label="View details"
            appearance="outlined"
            kind="secondary"
            onClick={() => router.push(pathCreator.unattachedSyncs({ envSlug }))}
          />
        </div>
      </div>
      <Card.Footer className="px-3">
        <div className="text-light text-sm">
          Last synced at <Time value={latestSyncTime} />
        </div>
      </Card.Footer>
    </Card>
  );
}
