'use client';

import { type Route } from 'next';
import { Link } from '@inngest/components/Link';
import { TextClickToCopy } from '@inngest/components/Text';
import { Time } from '@inngest/components/Time';
import { cn } from '@inngest/components/utils/classNames';

import { useEnvironment } from '@/components/Environments/environment-context';
import { FrameworkInfo } from '@/components/FrameworkInfo';
import { LanguageInfo } from '@/components/LanguageInfo';
import { SyncStatusPill } from '@/components/SyncStatusPill';
import { CardItem } from './CardItem';
import { PlatformSection } from './PlatformSection';

type Props = {
  // Optional because this card is used in the "unattached syncs" page, and
  // unattached syncs are by definition app-less
  app?: App;

  className?: string;
  sync: Sync | null;
  linkToSyncs?: boolean;
  loading?: false;
};

type LoadingProps = {
  app?: undefined;
  className?: string;
  sync?: undefined;
  linkToSyncs?: boolean;
  loading: true;
};

type App = {
  externalID: string;
  name: string;
};

type Sync = {
  framework: string | null;
  lastSyncedAt: Date;
  sdkLanguage: string | null;
  sdkVersion: string | null;
  status: string;
  url: string | null;
} & React.ComponentProps<typeof PlatformSection>['sync'];

export function AppInfoCard({ app, className, sync, linkToSyncs, loading }: Props | LoadingProps) {
  const env = useEnvironment();
  let lastSyncValue;
  if (sync) {
    if (app) {
      lastSyncValue = (
        <div className="flex items-center gap-2">
          <span className="hidden sm:block">
            <SyncStatusPill status={sync.status} />
          </span>
          <span className="block sm:hidden">
            <SyncStatusPill status={sync.status} iconOnly />
          </span>
          {linkToSyncs && <Time value={sync.lastSyncedAt} />}
          {!linkToSyncs && (
            <Link
              href={`/env/${env.slug}/apps/${encodeURIComponent(app.externalID)}/syncs` as Route}
              showIcon={false}
              internalNavigation
            >
              <Time value={sync.lastSyncedAt} />
            </Link>
          )}
        </div>
      );
    } else {
      lastSyncValue = (
        <div className="flex items-center gap-2">
          <span className="hidden sm:block">
            <SyncStatusPill status={sync.status} />
          </span>
          <span className="block sm:hidden">
            <SyncStatusPill status={sync.status} iconOnly />
          </span>
          <Time value={sync.lastSyncedAt} />
        </div>
      );
    }
  }

  return (
    <>
      <div
        className={cn('border-muted bg-canvasBase overflow-hidden rounded-lg border', className)}
      >
        <h2 className="border-muted text-basis border-b px-6 py-3 text-sm font-medium">
          App Information
        </h2>

        <dl className="flex flex-col gap-4 px-6 py-4 md:grid md:grid-cols-4">
          {/* Row 1 */}
          <CardItem
            detail={<div className="truncate">{app?.externalID ?? '-'}</div>}
            term="ID"
            loading={loading}
          />
          <CardItem
            detail={<div className="truncate">{sync?.sdkVersion ?? '-'}</div>}
            term="SDK Version"
            loading={loading}
          />
          <CardItem
            className="col-span-2"
            detail={<div className="truncate">{lastSyncValue ?? '-'}</div>}
            term="Last Sync"
            loading={loading}
          />

          {/* Row 2 */}
          <CardItem
            detail={<FrameworkInfo framework={sync?.framework} />}
            term="Framework"
            loading={loading}
          />
          <CardItem
            detail={<LanguageInfo language={sync?.sdkLanguage} />}
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
          {sync && <PlatformSection sync={sync} />}
        </dl>
      </div>
    </>
  );
}
