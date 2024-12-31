'use client';

import NextLink from 'next/link';
import { Link } from '@inngest/components/Link';
import { Time } from '@inngest/components/Time';
import { cn } from '@inngest/components/utils/classNames';
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
  'h-24 bg-canvasSubtle md:h-44 flex md:w-[410px] flex-col justify-center gap-2 px-10';
export const cardRightPanelStyles = 'h-44 flex-1 flex flex-col justify-center px-8';

export function UnattachedSyncsCard({ className, envSlug, latestSyncTime }: Props) {
  return (
    <div className={cn(cardWrapperStyles, className)}>
      <NextLink
        href={pathCreator.unattachedSyncs({ envSlug })}
        className={cn(cardLeftPanelStyles, 'hover:bg-canvasMuted transition-colors duration-300')}
      >
        <h2>
          <div className="text-basis flex items-center gap-1 font-medium">
            Unattached Syncs
            <RiArrowRightSLine className="h-4 w-4" />
          </div>
        </h2>
      </NextLink>
      <div className={cardRightPanelStyles}>
        <dl className="grid grid-cols-2 gap-4 min-[900px]:grid-cols-3">
          <p className="text-basis col-span-2 md:col-span-3">
            Unattached syncs are failed syncs that could not be associated with an app.
          </p>
          <Description
            className="col-span-2"
            detail={
              <div className="flex gap-2">
                <Link size="medium" href={pathCreator.unattachedSyncs({ envSlug })}>
                  <Time value={latestSyncTime} />
                </Link>
              </div>
            }
            term="Last sync"
          />
        </dl>
      </div>
    </div>
  );
}
