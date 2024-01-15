'use client';

import Image from 'next/image';
import Link from 'next/link';
import ArchiveBoxArrowDownIcon from '@heroicons/react/20/solid/ArchiveBoxArrowDownIcon';
import ChevronRightIcon from '@heroicons/react/20/solid/ChevronRightIcon';
import { Skeleton } from '@inngest/components/Skeleton';
import { classNames } from '@inngest/components/utils/classNames';

import { SyncStatus } from '@/components/SyncStatus';
import { Time } from '@/components/Time';
import AppDiagramImage from '@/images/app-diagram.png';

type Props = {
  app: App;
  className?: string;
  envSlug: string;
  isArchived?: boolean;
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

const cardWrapperStyles =
  'flex h-56 w-full min-w-[800px] max-w-[1200px] overflow-hidden rounded-lg border border-slate-300 bg-white';
const cardLeftPanelStyles = 'bg-slate-910 flex w-[410px] flex-col justify-center gap-2 px-10';

export function AppCard({ app, className, envSlug, isArchived }: Props) {
  return (
    <div className={classNames(cardWrapperStyles, className)}>
      <div className={cardLeftPanelStyles}>
        <h2>
          <Link
            className="transition-color flex cursor-pointer items-center gap-1 text-white underline decoration-transparent decoration-2 underline-offset-4 duration-300 hover:text-indigo-400 hover:decoration-indigo-400"
            href={`/env/${envSlug}/apps/${encodeURIComponent(app.externalID)}`}
          >
            {isArchived && <ArchiveBoxArrowDownIcon className="h-4 w-4" />}
            {app.name}
            <ChevronRightIcon className="h-4 w-4" />
          </Link>
        </h2>
        {app.latestSync?.url && (
          <dl>
            <dt className="hidden">URL</dt>
            <dd className="text-slate-400">{app.latestSync.url}</dd>
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
                  <SyncStatus status={app.latestSync.status} />
                  <Link
                    className="transition-color flex cursor-pointer items-center gap-1 text-indigo-400 underline decoration-transparent decoration-2 underline-offset-4 duration-300  hover:decoration-indigo-400"
                    href={`/env/${envSlug}/apps/${encodeURIComponent(app.externalID)}/syncs`}
                  >
                    <Time value={app.latestSync.createdAt} />
                    <ChevronRightIcon className="h-4 w-4" />
                  </Link>
                </div>
              )
            }
            term="Last sync"
          />
          <Description detail={app.latestSync?.sdkVersion} term="SDK Version" />

          {/* Row 2 */}
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

export function EmptyAppCard({ children }: { children: React.ReactNode }) {
  return (
    <div className={cardWrapperStyles}>
      <div className={classNames(cardLeftPanelStyles, 'overflow-hidden')}>
        <Image src={AppDiagramImage} alt="App diagram" />
      </div>
      <div className="flex flex-1 flex-col justify-center px-8">
        <p>
          Apps on Inngest act as clients for serving your functions.{' '}
          <span className="hidden lg:inline">
            In order to have your functions invoked by Inngest, you must sync your app.
          </span>{' '}
          Syncing is easy!
        </p>
        <ol className="mt-3 hidden flex-col gap-3 md:flex">
          <li className="flex items-center gap-2">
            <span className="h-6 w-6 rounded-full bg-slate-400 text-center text-white">1</span>
            <span className="flex-1">Deploy your app on your host environment of choice.</span>
          </li>
          <li className="flex items-center gap-2">
            <span className="h-6 w-6 rounded-full bg-slate-400 text-center text-white">2</span>
            <span className="flex-1">Sync with Inngest.</span>
          </li>
        </ol>
        {children}
      </div>
    </div>
  );
}

export function SkeletonCard() {
  return (
    <div className={cardWrapperStyles}>
      <div className={cardLeftPanelStyles} />
      <div className="flex flex-1 flex-col justify-center px-8">
        <Skeleton className="mb-2 block h-8 w-full" />
        <Skeleton className="mb-2 block h-8 w-full" />
        <Skeleton className="mb-2 block h-8 w-full" />
        <Skeleton className="mb-2 block h-8 w-full" />
      </div>
    </div>
  );
}
