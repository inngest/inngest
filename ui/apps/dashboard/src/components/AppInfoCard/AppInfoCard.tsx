'use client';

import Link from 'next/link';
import ChevronRightIcon from '@heroicons/react/20/solid/ChevronRightIcon';
import { classNames } from '@inngest/components/utils/classNames';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { FrameworkInfo } from '@/components/FrameworkInfo';
import { LanguageInfo } from '@/components/LanguageInfo';
import { SyncStatus } from '@/components/SyncStatus';
import { Time } from '@/components/Time';
import { PlatformSection } from './PlatformSection';

type Props = {
  app: App;
  className?: string;
  sync: Sync | null;
};

type App = {
  externalID: string;
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

export function AppInfoCard({ app, className, sync }: Props) {
  const env = useEnvironment();
  let lastSyncValue;
  if (sync) {
    lastSyncValue = (
      <div className="flex gap-2">
        <SyncStatus status={sync?.status} />
        <Link
          className="transition-color flex cursor-pointer items-center gap-1 text-indigo-400 underline decoration-transparent decoration-2 underline-offset-4 duration-300  hover:decoration-indigo-400"
          href={`/env/${env.slug}/apps/${encodeURIComponent(app.externalID)}/syncs`}
        >
          <Time value={sync?.createdAt} />
          <ChevronRightIcon className="h-4 w-4" />
        </Link>
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
        <h2 className="border-b border-slate-300 px-6 py-3 text-sm font-medium text-slate-600">
          App Information
        </h2>

        <dl className="grid grow grid-cols-4 gap-4 px-6 py-4">
          {/* Row 1 */}
          <Description className="truncate" detail={app.externalID} term="ID" />
          <Description className="truncate" detail={sync?.sdkVersion ?? '-'} term="SDK Version" />
          <Description
            className="col-span-2 truncate"
            detail={lastSyncValue ?? '-'}
            term="Last Sync"
          />

          {/* Row 2 */}
          <Description
            className="truncate"
            detail={<FrameworkInfo framework={sync?.framework} />}
            term="Framework"
          />
          <Description
            className="truncate"
            detail={<LanguageInfo language={sync?.sdkLanguage} />}
            term="Language"
          />
          <Description className="col-span-2 truncate" detail={sync?.url ?? '-'} term="URL" />

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
}: {
  className?: string;
  detail: React.ReactNode;
  term: string;
}) {
  return (
    <div className={className}>
      <dt className="pb-2 text-sm text-slate-400">{term}</dt>
      <dd className="text-slate-800">{detail}</dd>
    </div>
  );
}
