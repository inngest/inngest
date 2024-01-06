'use client';

import Link from 'next/link';
import ChevronRightIcon from '@heroicons/react/20/solid/ChevronRightIcon';
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
  functionCount: number;
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
  url: string | null;
};

export function AppCard({ app, className, envSlug }: Props) {
  return (
    <div
      className={classNames(
        'flex h-56 w-full min-w-[800px] max-w-[1200px] overflow-hidden rounded-lg border border-slate-300 bg-white',
        className
      )}
    >
      <div className="bg-slate-910 flex w-[410px] flex-col justify-center gap-2 px-10">
        <h2>
          <Link
            className="transition-color flex cursor-pointer items-center gap-1 text-white underline decoration-transparent decoration-2 underline-offset-4 duration-300 hover:text-indigo-400 hover:decoration-indigo-400"
            href={`/env/${envSlug}/apps/${encodeURIComponent(app.externalID)}`}
          >
            {app.name}
            <ChevronRightIcon className="h-4 w-4" />
          </Link>
        </h2>
        {app.latestSync?.url && (
          <dl>
            <dt className="hidden">URL</dt>
            <dd className="text-slate-400">{app.latestSync?.url}</dd>
          </dl>
        )}
      </div>
      <div className="flex flex-1 items-center px-8">
        <dl className="grid grow grid-cols-2 gap-4 md:grid-cols-3">
          {/* Row 1 */}
          <Description
            className="col-span-2"
            detail={
              app.latestSync && (
                <div className="flex gap-2">
                  <SyncStatus status={app.latestSync?.status} />
                  <Link
                    className="transition-color flex cursor-pointer items-center gap-1 text-indigo-400 underline decoration-transparent decoration-2 underline-offset-4 duration-300  hover:decoration-indigo-400"
                    href={`/env/${envSlug}/apps/${encodeURIComponent(app.externalID)}/syncs`}
                  >
                    <Time value={app.latestSync?.createdAt} />
                    <ChevronRightIcon className="h-4 w-4" />
                  </Link>
                </div>
              )
            }
            term="App ID"
          />
          <Description detail={app.latestSync?.sdkVersion} term="SDK Version" />

          {/* Row 2 */}
          {/* <Description
          detail={<LanguageInfo language={app.latestSync?.sdkLanguage} />}
          term="Language"
        />
        <Description
          detail={<FrameworkInfo framework={app.latestSync?.framework} />}
          term="Framework"
        />
        <Description
          detail={<PlatformInfo platform={app.latestSync?.platform} />}
          term="Platform"
        /> */}

          {/* Row 3 */}

          <Description detail={`${app.functionCount} Functions`} term="Functions" />
        </dl>
      </div>
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
      <dt className="pb-2 text-sm text-slate-400">{term}</dt>
      <dd className="text-slate-800">{detail ?? ''}</dd>
    </div>
  );
}
