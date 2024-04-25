'use client';

import Link from 'next/link';
import { Link as InngestLink } from '@inngest/components/Link';
import { Time } from '@inngest/components/Time';
import { classNames } from '@inngest/components/utils/classNames';
import { RiArrowRightSLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';
import { cardWrapperStyles } from './AppCard';
import { Description } from './Description';

type Props = {
  className?: string;
  envSlug: string;
  latestSyncTime: Date;
};

export const cardLeftPanelStyles =
  'h-24 bg-slate-500 md:h-44 flex md:w-[410px] flex-col justify-center gap-2 px-10';
export const cardRightPanelStyles = 'h-44 flex-1 flex flex-col justify-center px-8';

export function UnattachedSyncsCard({ className, envSlug, latestSyncTime }: Props) {
  return (
    <div className={classNames(cardWrapperStyles, className)}>
      <div className={cardLeftPanelStyles}>
        <h2>
          <Link
            className="transition-color flex cursor-pointer items-center gap-1 text-white underline decoration-transparent decoration-2 underline-offset-4 duration-300 hover:text-indigo-300 hover:decoration-indigo-300"
            href={pathCreator.unattachedSyncs({ envSlug })}
          >
            Unattached Syncs
            <RiArrowRightSLine className="h-4 w-4" />
          </Link>
        </h2>
      </div>
      <div className={cardRightPanelStyles}>
        <dl className="grid grid-cols-2 gap-4 min-[900px]:grid-cols-3">
          <p className="col-span-2 md:col-span-3">
            Unattached syncs are failed syncs that could not be associated with an app.
          </p>
          <Description
            className="col-span-2"
            detail={
              <div className="flex gap-2">
                <InngestLink
                  internalNavigation
                  showIcon={false}
                  href={pathCreator.unattachedSyncs({ envSlug })}
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
