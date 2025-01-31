'use client';

import { useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { AppCard } from '@inngest/components/Apps/AppCard';
import { Button } from '@inngest/components/Button/Button';
import { Pill } from '@inngest/components/Pill/Pill';

import { type FlattenedApp } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/apps/useApps';
import getAppCardContent from '@/components/Apps/AppCardContent';
import { pathCreator } from '@/utils/urls';

export default function AppCards({ apps, envSlug }: { apps: FlattenedApp[]; envSlug: string }) {
  const router = useRouter();
  const sortedApps = useMemo(() => {
    return [...apps].sort((a, b) => {
      return (
        (b.lastSyncedAt ? new Date(b.lastSyncedAt).getTime() : 0) -
        (a.lastSyncedAt ? new Date(a.lastSyncedAt).getTime() : 0)
      );
    });
  }, [apps]);

  return sortedApps.map((app) => {
    const { appKind, status, footerHeaderTitle, footerHeaderSecondaryCTA, footerContent } =
      getAppCardContent({
        app,
        envSlug,
      });

    return (
      <div className="mb-6" key={app.id}>
        <AppCard kind={appKind}>
          <AppCard.Content
            app={app}
            pill={
              status ? (
                <Pill appearance="outlined" kind={appKind}>
                  {status}
                </Pill>
              ) : null
            }
            actions={
              <div className="flex items-center gap-2">
                <Button
                  appearance="outlined"
                  label="View details"
                  onClick={() =>
                    router.push(pathCreator.app({ envSlug, externalAppID: app.externalID }))
                  }
                />
              </div>
            }
          />
          <AppCard.Footer
            kind={appKind}
            headerTitle={footerHeaderTitle}
            headerSecondaryCTA={footerHeaderSecondaryCTA}
            content={footerContent}
          />
        </AppCard>
      </div>
    );
  });
}
