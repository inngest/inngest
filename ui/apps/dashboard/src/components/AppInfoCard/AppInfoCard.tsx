'use client';

import { type Route } from 'next';
import { Link } from '@inngest/components/Link';
import { Skeleton } from '@inngest/components/Skeleton';
import { classNames } from '@inngest/components/utils/classNames';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { FrameworkInfo } from '@/components/FrameworkInfo';
import { LanguageInfo } from '@/components/LanguageInfo';
import { SyncStatus } from '@/components/SyncStatus';
import { Time } from '@/components/Time';
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
          <SyncStatus status={sync.status} />
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
          <SyncStatus status={sync.status} />
          <Time value={sync.lastSyncedAt} />
        </div>
      );
    }
  }

  return (
    <>
      <div
        className={classNames(
          'overflow-hidden rounded-lg border border-slate-300 bg-white',
          className
        )}
      >
        <h2 className="border-b border-slate-300 px-6 py-3 text-sm font-medium text-slate-600">
          App Information
        </h2>

        <dl className="grid grow grid-cols-4 gap-4 px-6 py-4">
          {/* Row 1 */}
          <Description
            className="truncate"
            detail={app?.externalID ?? '-'}
            term="ID"
            loading={loading}
          />
          <Description
            className="truncate"
            detail={sync?.sdkVersion ?? '-'}
            term="SDK Version"
            loading={loading}
          />
          <Description
            className="col-span-2 truncate"
            detail={lastSyncValue ?? '-'}
            term="Last Sync"
            loading={loading}
          />

          {/* Row 2 */}
          <Description
            className="truncate"
            detail={<FrameworkInfo framework={sync?.framework} />}
            term="Framework"
            loading={loading}
          />
          <Description
            className="truncate"
            detail={<LanguageInfo language={sync?.sdkLanguage} />}
            term="Language"
            loading={loading}
          />
          <Description
            className="col-span-2 truncate"
            detail={sync?.url ?? '-'}
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

function Description({
  className,
  detail,
  term,
  loading,
}: {
  className?: string;
  detail: React.ReactNode;
  term: string;
  loading?: boolean;
}) {
  return (
    <div className={className}>
      <dt className="pb-2 text-sm text-slate-400">{term}</dt>
      {!loading && <dd className="text-slate-800">{detail}</dd>}
      {loading && <Skeleton className="mb-3.5 block h-6 w-full" />}
    </div>
  );
}
