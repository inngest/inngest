'use client';

import Link from 'next/link';
import type { Function } from '@inngest/components/types/function';
import { classNames } from '@inngest/components/utils/classNames';

import { FrameworkInfo } from '@/components/FrameworkInfo';
import { LanguageInfo } from '@/components/LanguageInfo';
import { PlatformInfo } from '@/components/PlatformInfo';
import { SyncStatus } from '@/components/SyncStatus';
import { Time } from '@/components/Time';

type Props = {
  app: App;
  className?: string;
  envSlug: string;
};

type App = {
  externalID: string;
  latestSync: Sync | null;
  name: string;
};

type Sync = {
  createdAt: Date;
  framework: string | null;
  platform: string | null;
  sdkLanguage: string | null;
  sdkVersion: string | null;
  status: string;
  syncedFunctions: Pick<Function, 'id'>[];
  url: string | null;
};

export function AppCard({ app, className, envSlug }: Props) {
  return (
    <div
      className={classNames(
        'flex w-full min-w-[800px] max-w-[1200px] overflow-hidden rounded-lg border border-slate-300 bg-white',
        className
      )}
    >
      <div className="m-4 mt-8 flex w-64 items-center border-r border-slate-400">
        <h2>
          <Link
            className="text-indigo-600 hover:underline"
            href={`/env/${envSlug}/apps/${encodeURIComponent(app.externalID)}`}
          >
            {app.name}
          </Link>
        </h2>
      </div>

      <dl className="m-4 grid grow grid-cols-3 gap-4">
        {/* Row 1 */}
        <Description detail={app.externalID} term="App ID" />
        <Description
          className="col-span-2"
          detail={
            app.latestSync && (
              <div className="flex gap-2">
                <Time value={app.latestSync?.createdAt} />
                <SyncStatus status={app.latestSync?.status} />
              </div>
            )
          }
          term="App ID"
        />

        {/* Row 2 */}
        <Description detail={app.latestSync?.sdkVersion} term="SDK Version" />
        <Description
          detail={<LanguageInfo language={app.latestSync?.sdkLanguage} />}
          term="Language"
        />
        <Description
          detail={<FrameworkInfo framework={app.latestSync?.framework} />}
          term="Framework"
        />

        {/* Row 3 */}
        <Description
          detail={<PlatformInfo platform={app.latestSync?.platform} />}
          term="Platform"
        />
        <Description detail={app.latestSync?.syncedFunctions.length} term="Functions" />

        {/* Row 4 */}
        <Description className="col-span-3" detail={app.latestSync?.url} term="URL" />
      </dl>
    </div>
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
      <dd>{detail ?? ''}</dd>
    </div>
  );
}
