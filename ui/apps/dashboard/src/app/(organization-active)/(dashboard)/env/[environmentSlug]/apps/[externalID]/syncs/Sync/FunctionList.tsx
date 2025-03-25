import { useMemo } from 'react';
import NextLink from 'next/link';
import { Button } from '@inngest/components/Button';
import { defaultLinkStyles } from '@inngest/components/Link';
import type { Function } from '@inngest/components/types/function';
import { cn } from '@inngest/components/utils/classNames';
import { RiArrowDownSLine, RiArrowRightLine } from '@remixicon/react';
import { useLocalStorage } from 'react-use';

import {
  CollapsibleCardContent,
  CollapsibleCardContentWrapper,
  CollapsibleCardHeader,
  CollapsibleCardItem,
  CollapsibleCardRoot,
  CollapsibleCardTrigger,
} from '@/components/CollapsibleCard';
import { useEnvironment } from '@/components/Environments/environment-context';
import { pathCreator } from '@/utils/urls';

type Fn = Pick<Function, 'id' | 'name' | 'slug'>;

type Props = {
  className?: string;
  removedFunctions: Fn[];
  syncedFunctions: Fn[];
};

export function FunctionList({ removedFunctions, syncedFunctions }: Props) {
  const env = useEnvironment();
  const [isSyncedFunctionsCardOpen, setIsSyncedFunctionsCardOpen] = useLocalStorage(
    'AppSyncedFunctionsOpened',
    true
  );
  const [isRemovedFunctionsCardOpen, setIsRemovedFunctionsCardOpen] = useLocalStorage(
    'AppRemovedFunctionsOpened',
    true
  );

  removedFunctions = useMemo(() => {
    return [...removedFunctions].sort((a, b) => {
      return a.name.localeCompare(b.name);
    });
  }, [removedFunctions]);

  syncedFunctions = useMemo(() => {
    return [...syncedFunctions].sort((a, b) => {
      return a.name.localeCompare(b.name);
    });
  }, [syncedFunctions]);

  return (
    <div className="flex flex-col gap-4">
      <CollapsibleCardRoot
        type="single"
        defaultValue={isSyncedFunctionsCardOpen ? 'syncedFunctions' : undefined}
        collapsible
      >
        <CollapsibleCardItem value="syncedFunctions">
          <CollapsibleCardHeader className="data-[state=open]:border-subtle text-basis flex h-11 items-center justify-between border-b border-transparent px-6 text-sm font-medium">
            <p>Synced functions ({syncedFunctions.length})</p>
            <CollapsibleCardTrigger
              asChild
              onClick={() => setIsSyncedFunctionsCardOpen(!isSyncedFunctionsCardOpen)}
            >
              <Button
                className="group"
                appearance="outlined"
                kind="secondary"
                icon={
                  <RiArrowDownSLine className="transform-90 transition-transform duration-500 group-data-[state=open]:-rotate-180" />
                }
              />
            </CollapsibleCardTrigger>
          </CollapsibleCardHeader>
          <CollapsibleCardContentWrapper>
            {isSyncedFunctionsCardOpen && (
              <CollapsibleCardContent>
                {syncedFunctions.map((fn, i) => {
                  const isLast = i === syncedFunctions.length - 1;

                  return (
                    <NextLink
                      href={pathCreator.function({ envSlug: env.slug, functionSlug: fn.slug })}
                      key={fn.id}
                    >
                      <div
                        className={cn(
                          defaultLinkStyles,
                          'border-subtle hover:bg-canvasSubtle/50 group flex w-full items-center gap-2 py-3 pl-4 pr-2 text-sm font-medium',
                          !isLast && 'border-b'
                        )}
                      >
                        {fn.name}
                        <RiArrowRightLine className="h-3 w-3 -translate-x-3 opacity-0 transition-all group-hover:translate-x-0 group-hover:opacity-100" />
                      </div>
                    </NextLink>
                  );
                })}
                {syncedFunctions.length === 0 && (
                  <div className="text-subtle p-2 text-center text-sm">No synced functions</div>
                )}
              </CollapsibleCardContent>
            )}
          </CollapsibleCardContentWrapper>
        </CollapsibleCardItem>
      </CollapsibleCardRoot>
      <CollapsibleCardRoot
        type="single"
        defaultValue={isRemovedFunctionsCardOpen ? 'RemovedFunctions' : undefined}
        collapsible
      >
        <CollapsibleCardItem value="RemovedFunctions">
          <CollapsibleCardHeader className="data-[state=open]:border-subtle text-basis flex h-11 items-center justify-between border-b border-transparent px-6 text-sm font-medium">
            <p>Removed functions ({removedFunctions.length})</p>
            <CollapsibleCardTrigger
              asChild
              onClick={() => setIsRemovedFunctionsCardOpen(!isRemovedFunctionsCardOpen)}
            >
              <Button
                className="group"
                appearance="outlined"
                kind="secondary"
                icon={
                  <RiArrowDownSLine className="transform-90 transition-transform duration-500 group-data-[state=open]:-rotate-180" />
                }
              />
            </CollapsibleCardTrigger>
          </CollapsibleCardHeader>
          {isRemovedFunctionsCardOpen && (
            <CollapsibleCardContent>
              {removedFunctions.map((fn, i) => {
                const isLast = i === removedFunctions.length - 1;

                return (
                  <NextLink
                    href={pathCreator.function({ envSlug: env.slug, functionSlug: fn.slug })}
                    key={fn.id}
                  >
                    <div
                      className={cn(
                        defaultLinkStyles,
                        'border-subtle hover:bg-canvasSubtle/50 group flex w-full items-center gap-2 py-3 pl-4 pr-2 text-sm font-medium',
                        !isLast && 'border-b'
                      )}
                    >
                      {fn.name}
                      <RiArrowRightLine className="h-3 w-3 -translate-x-3 opacity-0 transition-all group-hover:translate-x-0 group-hover:opacity-100" />
                    </div>
                  </NextLink>
                );
              })}

              {removedFunctions.length === 0 && (
                <div className="text-subtle p-2 text-center text-sm">No removed functions</div>
              )}
            </CollapsibleCardContent>
          )}
        </CollapsibleCardItem>
      </CollapsibleCardRoot>
    </div>
  );
}
