'use client';

import { type Route } from 'next';
import { AppDetailsCard, CardItem } from '@inngest/components/Apps/AppDetailsCard';
import { Link } from '@inngest/components/Link';
import { Pill } from '@inngest/components/Pill/Pill';
import { TextClickToCopy } from '@inngest/components/Text';
import { Time } from '@inngest/components/Time';
import { methodTypes, type App } from '@inngest/components/types/app';
import { RiArrowLeftRightLine, RiInfinityLine } from '@remixicon/react';

import { useEnvironment } from '@/components/Environments/environment-context';
import { SyncStatusPill } from '@/components/SyncStatusPill';
import { PlatformSection } from './PlatformSection';

type Props = {
  // Optional because this card is used in the "unattached syncs" page, and
  // unattached syncs are by definition app-less
  app?: App;

  className?: string;
  sync: Sync | null;
  linkToSyncs?: boolean;
  loading?: false;
  workerCounter?: React.ReactNode;
};

type LoadingProps = {
  app?: undefined;
  className?: string;
  sync?: undefined;
  linkToSyncs?: boolean;
  loading: true;
  workerCounter?: React.ReactNode;
};

type Sync = {
  framework?: string | null;
  lastSyncedAt: Date;
  sdkLanguage?: string | null;
  sdkVersion: string | null;
  status: string;
  url: string | null;
} & React.ComponentProps<typeof PlatformSection>['sync'];

export function AppInfoCard({
  app,
  className,
  sync,
  linkToSyncs,
  loading,
  workerCounter,
}: Props | LoadingProps) {
  const env = useEnvironment();
  let lastSyncValue;
  if (sync) {
    if (app) {
      lastSyncValue = (
        <div className="flex items-center gap-2">
          <SyncStatusPill status={sync.status} />
          {linkToSyncs && <Time value={sync.lastSyncedAt} />}
          {!linkToSyncs && app.externalID && (
            <Link
              href={`/env/${env.slug}/apps/${encodeURIComponent(app.externalID)}/syncs` as Route}
              size="small"
            >
              <Time value={sync.lastSyncedAt} />
            </Link>
          )}
        </div>
      );
    } else {
      lastSyncValue = (
        <div className="flex items-center gap-2">
          <SyncStatusPill status={sync.status} />
          <Time value={sync.lastSyncedAt} />
        </div>
      );
    }
  }

  return (
    <>
      <AppDetailsCard title="App information" className={className}>
        {/* Row 1 */}
        <CardItem
          detail={<div className="truncate">{app?.externalID ?? '-'}</div>}
          term="App ID"
          loading={loading}
        />
        <CardItem
          detail={
            <div className="truncate">
              {sync?.sdkVersion ? <Pill>{sync.sdkVersion}</Pill> : '-'}
            </div>
          }
          term="SDK version"
          loading={loading}
        />
        <CardItem
          className="col-span-2"
          detail={<div className="truncate">{lastSyncValue ?? '-'}</div>}
          term="Last sync"
          loading={loading}
        />

        {/* Row 2 */}
        <CardItem
          detail={<div className="truncate">{sync?.framework ?? '-'}</div>}
          term="Framework"
          loading={loading}
        />
        <CardItem
          detail={<div className="truncate">{sync?.sdkLanguage || '-'}</div>}
          term="Language"
          loading={loading}
        />
        <CardItem
          className="col-span-2"
          detail={<TextClickToCopy truncate>{sync?.url ?? '-'}</TextClickToCopy>}
          term="URL"
          loading={loading}
        />
        {/* Row 3 */}
        {app?.method && (
          <CardItem
            term="Method"
            detail={
              <div className="flex items-center gap-1">
                {app.method === methodTypes.Connect ? (
                  <RiInfinityLine className="h-4 w-4" />
                ) : (
                  <RiArrowLeftRightLine className="h-4 w-4" />
                )}
                <div className="lowercase first-letter:capitalize">{app.method}</div>
              </div>
            }
          />
        )}
        <CardItem
          detail={<div className="truncate">{app?.version ? <Pill>{app.version}</Pill> : '-'}</div>}
          term="App version"
          loading={loading}
        />
        {app?.method === methodTypes.Connect && workerCounter && <>{workerCounter}</>}

        {/* Row 4 */}
        {sync && <PlatformSection sync={sync} />}
      </AppDetailsCard>
    </>
  );
}
