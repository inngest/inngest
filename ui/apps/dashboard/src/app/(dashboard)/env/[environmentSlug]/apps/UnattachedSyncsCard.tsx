'use client';

import { type Route } from 'next';
import Link from 'next/link';
import ChevronRightIcon from '@heroicons/react/20/solid/ChevronRightIcon';
import { Link as InngestLink } from '@inngest/components/Link';
import { classNames } from '@inngest/components/utils/classNames';

import { Time } from '@/components/Time';
import { Description } from './Description';

type Props = {
  className?: string;
  envSlug: string;
  latestSyncTime: Date;
};

const cardWrapperStyles =
  'flex w-full min-w-[800px] max-w-[1200px] overflow-hidden rounded-lg border border-slate-300 bg-white';
const cardLeftPanelStyles =
  'bg-slate-200 flex w-[410px] flex-col justify-center gap-2 px-10 border-r border-slate-300';

export function UnattachedSyncsCard({ className, envSlug, latestSyncTime }: Props) {
  return (
    <div className={classNames(cardWrapperStyles, className)}>
      <div className={cardLeftPanelStyles}>
        <h2>
          <Link
            className="transition-color flex cursor-pointer items-center gap-1 underline decoration-transparent decoration-2 underline-offset-4 duration-300 hover:text-indigo-400 hover:decoration-indigo-400"
            href={`/env/${envSlug}/unattached-syncs`}
          >
            Unattached Syncs
            <ChevronRightIcon className="h-4 w-4" />
          </Link>
        </h2>
      </div>
      <div className="flex flex-1 items-center px-8 py-4">
        <dl className="grid grow grid-cols-2 gap-4 md:grid-cols-3">
          <Description
            className="col-span-2"
            detail={
              <div className="flex gap-2">
                <InngestLink
                  internalNavigation
                  showIcon={false}
                  href={`/env/${envSlug}/unattached-syncs` as Route}
                >
                  <Time value={latestSyncTime} />
                </InngestLink>
              </div>
            }
            term="Last sync"
          />
        </dl>
      </div>
    </div>
  );
}
