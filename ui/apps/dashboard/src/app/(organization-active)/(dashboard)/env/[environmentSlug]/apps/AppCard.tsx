'use client';

import Image from 'next/image';
import Link from 'next/link';
import { Link as InngestLink } from '@inngest/components/Link';
import { Skeleton } from '@inngest/components/Skeleton';
import { Time } from '@inngest/components/Time';
import { cn } from '@inngest/components/utils/classNames';
import { RiArchive2Line, RiArrowRightSLine } from '@remixicon/react';

import { SyncStatusPill } from '@/components/SyncStatusPill';
import AppDiagramImage from '@/images/app-diagram.png';
import { pathCreator } from '@/utils/urls';

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
  error: string | null;
  framework: string | null;
  lastSyncedAt: Date;
  platform: string | null;
  sdkLanguage: string | null;
  sdkVersion: string | null;
  status: string;
  url: string | null;
};

export const cardWrapperStyles =
  'md:flex w-full lg:min-w-[800px] max-w-[1200px] overflow-hidden rounded-lg border border-subtle bg-canvasBase';
const cardLeftPanelStyles =
  'h-36 md:h-56 bg-canvasSubtle flex md:w-[410px] flex-col justify-center gap-2 px-10';
const cardRightPanelStyles = 'h-56 flex flex-col justify-center px-8';

export function AppCard({ app, className, envSlug, isArchived }: Props) {
  const latestSyncURL = app.latestSync?.url?.replace(/^https:\/\//, '').replace(/\?.+$/, '');
  return (
    <div className={cn(cardWrapperStyles, className)}>
      <Link
        href={pathCreator.app({ envSlug, externalAppID: app.externalID })}
        className={cn(cardLeftPanelStyles, 'hover:bg-canvasMuted')}
      >
        <h2>
          <div className="text-basis flex items-center gap-1 font-medium">
            {isArchived && <RiArchive2Line className="h-4 w-4" />}
            <span className="truncate" title={app.name}>
              {app.name}
            </span>
            <RiArrowRightSLine className="h-4 w-4" />
          </div>
        </h2>
        {latestSyncURL && (
          <dl>
            <dt className="hidden">URL</dt>
            <dd className="text-subtle truncate" title={app.latestSync?.url || ''}>
              {latestSyncURL}
            </dd>
          </dl>
        )}
      </Link>
      <div className="flex h-56 flex-1 flex-col">
        {app.latestSync?.error && (
          <div className="bg-red-100 px-8 py-2 text-red-800">{app.latestSync.error}</div>
        )}

        <div className={cn(cardRightPanelStyles, 'h-full')}>
          <dl className="grid grid-cols-2 gap-4 min-[900px]:grid-cols-3">
            {/* Row 1 */}
            <Description
              className="col-span-2"
              detail={
                app.latestSync && (
                  <div className="flex items-center gap-2">
                    <SyncStatusPill status={app.latestSync.status} />
                    <InngestLink
                      internalNavigation
                      showIcon={false}
                      href={pathCreator.appSyncs({ envSlug, externalAppID: app.externalID })}
                    >
                      <Time value={app.latestSync.lastSyncedAt} />
                    </InngestLink>
                  </div>
                )
              }
              term="Last sync"
            />
            <Description detail={app.latestSync?.sdkVersion} term="SDK Version" />

            {/* Row 2 */}
            <Description
              detail={`${app.functionCount} ${app.functionCount === 1 ? 'Function' : 'Functions'}`}
              term="Functions"
            />
          </dl>
        </div>
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
      <dt className="text-subtle pb-2 text-sm">{term}</dt>
      <dd className="text-basis leading-8">{detail ?? ''}</dd>
    </div>
  );
}

export function EmptyAppCard({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={cn(cardWrapperStyles, className)}>
      <div className={cn(cardLeftPanelStyles, 'bg-slate-910 items-center overflow-hidden')}>
        <Image src={AppDiagramImage} alt="App diagram" className="object-none md:object-fill" />
      </div>
      <div className={cn(cardRightPanelStyles, 'flex-1')}>
        <p>
          When you serve your functions using our serve API handler, you are hosting a new Inngest
          app.{' '}
          <span className="hidden lg:inline">
            In order to have your functions invoked by Inngest, you must sync your app.
          </span>{' '}
          Syncing is easy!
        </p>
        <ol className="mt-3 flex flex-col gap-3">
          <li className="flex items-center gap-2">
            <span className="h-6 w-6 rounded-full bg-slate-400 text-center text-white">1</span>
            <span className="flex-1">Deploy your code to your hosted platform of choice.</span>
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
      <div className={cn(cardRightPanelStyles, 'flex-1')}>
        <Skeleton className="mb-2 block h-8 w-full" />
        <Skeleton className="mb-2 block h-8 w-full" />
        <Skeleton className="mb-2 block h-8 w-full" />
        <Skeleton className="mb-2 block h-8 w-full" />
      </div>
    </div>
  );
}
