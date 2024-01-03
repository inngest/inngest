'use client';

import { classNames } from '@inngest/components/utils/classNames';

import { FrameworkInfo } from '@/components/FrameworkInfo';
import { LanguageInfo } from '@/components/LanguageInfo';
import { SyncStatus } from '@/components/SyncStatus';
import { Time } from '@/components/Time';
import { PlatformSection } from './PlatformSection';

type Props = {
  app: App;
  className?: string;
};

type App = {
  externalID: string;
  latestSync: Sync | null;
  name: string;
};

type Sync = {
  createdAt: Date;
  framework: string | null;
  sdkLanguage: string | null;
  sdkVersion: string | null;
  status: string;
  url: string | null;
} & React.ComponentProps<typeof PlatformSection>['sync'];

export function AppInfoCard({ app, className }: Props) {
  let lastSyncValue;
  if (app.latestSync) {
    lastSyncValue = (
      <div className="flex gap-2">
        <Time value={app.latestSync.createdAt} />
        <SyncStatus status={app.latestSync.status} />
      </div>
    );
  }

  return (
    <>
      <div
        className={classNames(
          'overflow-hidden rounded-lg border border-slate-300 bg-white',
          className
        )}
      >
        <div className="border-b border-slate-300 px-4 py-2">App Info</div>

        <dl className="grid grow grid-cols-4 gap-4 p-4">
          {/* Row 1 */}
          <Description className="truncate" detail={app.externalID} term="ID" />
          <Description
            className="truncate"
            detail={app.latestSync?.sdkVersion ?? '-'}
            term="SDK Version"
          />
          <Description
            className="col-span-2 truncate"
            detail={lastSyncValue ?? '-'}
            term="Last Sync"
          />

          {/* Row 2 */}
          <Description
            className="truncate"
            detail={<FrameworkInfo framework={app.latestSync?.framework} />}
            term="Framework"
          />
          <Description
            className="truncate"
            detail={<LanguageInfo language={app.latestSync?.sdkLanguage} />}
            term="Language"
          />
          <Description
            className="col-span-2 truncate"
            detail={app.latestSync?.url ?? '-'}
            term="URL"
          />

          {/* Row 3 */}
          {app.latestSync && <PlatformSection sync={app.latestSync} />}
        </dl>
      </div>
    </>
  );
}

function Description({
  className,
  detail,
  term,
}: {
  className?: string;
  detail: React.ReactNode;
  term: string;
}) {
  return (
    <div className={className}>
      <dt className="text-xs text-slate-600">{term}</dt>
      <dd>{detail}</dd>
    </div>
  );
}
