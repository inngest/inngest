'use client';

import type { Function } from '@inngest/components/types/function';
import { classNames } from '@inngest/components/utils/classNames';

import { FrameworkInfo } from '@/components/FrameworkInfo';
import { Labeled } from '@/components/Labeled';
import { LanguageInfo } from '@/components/LanguageInfo';
import { PlatformInfo } from '@/components/PlatformInfo';
import { SyncStatus } from '@/components/SyncStatus';
import { Time } from '@/components/Time';

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
  platform: string | null;
  sdkLanguage: string | null;
  sdkVersion: string | null;
  status: string;
  syncedFunctions: Pick<Function, 'id'>[];
  url: string | null;
};

export function AppCard({ app, className }: Props) {
  return (
    <div
      className={classNames(
        'flex w-full min-w-[800px] max-w-[1200px] overflow-hidden rounded-lg border border-slate-300 bg-white',
        className
      )}
    >
      <div className="m-4 mt-8 flex w-64 items-center border-r border-slate-400">
        <h2>
          {/* TODO: Make this a link */}
          {app.name}
        </h2>
      </div>

      <div className="m-4 grid grow grid-cols-3 gap-4">
        {/* Row 1 */}
        <Labeled label="App ID" value={app.externalID} />
        <Labeled
          className="col-span-2"
          label="Last Sync"
          value={
            app.latestSync && (
              <div className="flex gap-2">
                <Time value={app.latestSync?.createdAt} />
                <SyncStatus status={app.latestSync?.status} />
              </div>
            )
          }
        />

        {/* Row 2 */}
        <Labeled label="SDK Version" value={app.latestSync?.sdkVersion} />
        <Labeled label="Language" value={<LanguageInfo language={app.latestSync?.sdkLanguage} />} />
        <Labeled
          label="Framework"
          value={<FrameworkInfo framework={app.latestSync?.framework} />}
        />

        {/* Row 3 */}
        <Labeled label="Platform" value={<PlatformInfo platform={app.latestSync?.platform} />} />
        <Labeled label="Functions" value={app.latestSync?.syncedFunctions.length} />

        {/* Row 4 */}
        <Labeled className="col-span-3" label="URL" value={app.latestSync?.url} />
      </div>
    </div>
  );
}
