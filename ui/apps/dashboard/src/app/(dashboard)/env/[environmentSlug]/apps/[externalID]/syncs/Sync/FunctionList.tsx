import { useMemo } from 'react';
import Link from 'next/link';
import ArrowRightIcon from '@heroicons/react/20/solid/ArrowRightIcon';
import ChevronDownIcon from '@heroicons/react/20/solid/ChevronDownIcon';
import { Button } from '@inngest/components/Button';
import { defaultLinkStyles } from '@inngest/components/Link';
import type { Function } from '@inngest/components/types/function';
import { classNames } from '@inngest/components/utils/classNames';
import { useLocalStorage } from 'react-use';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import {
  CollapsibleCardContent,
  CollapsibleCardContentWrapper,
  CollapsibleCardHeader,
  CollapsibleCardItem,
  CollapsibleCardRoot,
  CollapsibleCardTrigger,
} from '@/components/CollapsibleCard';
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
          <CollapsibleCardHeader className="flex h-11 items-center justify-between border-b border-transparent px-6 text-sm font-medium text-slate-600 data-[state=open]:border-slate-300">
            <p>Synced Functions ({syncedFunctions.length})</p>
            <CollapsibleCardTrigger
              asChild
              onClick={() => setIsSyncedFunctionsCardOpen(!isSyncedFunctionsCardOpen)}
            >
              <Button
                className="group"
                appearance="outlined"
                icon={
                  <ChevronDownIcon className="transform-90 text-slate-500 transition-transform duration-500 group-data-[state=open]:-rotate-180" />
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
                    <Link
                      href={pathCreator.function({ envSlug: env.slug, functionSlug: fn.slug })}
                      key={fn.id}
                    >
                      <div
                        className={classNames(
                          defaultLinkStyles,
                          'group flex w-full items-center gap-2 border-slate-200 py-3 pl-6 pr-2 text-sm font-medium hover:bg-slate-100',
                          !isLast && 'border-b'
                        )}
                      >
                        {fn.name}
                        <ArrowRightIcon className="h-3 w-3 -translate-x-3 opacity-0 transition-all group-hover:translate-x-0 group-hover:opacity-100" />
                      </div>
                    </Link>
                  );
                })}
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
          <CollapsibleCardHeader className="flex h-11 items-center justify-between border-b border-transparent px-6 text-sm font-medium text-slate-600 data-[state=open]:border-slate-300">
            <p>Removed Functions ({removedFunctions.length})</p>
            <CollapsibleCardTrigger
              asChild
              onClick={() => setIsRemovedFunctionsCardOpen(!isRemovedFunctionsCardOpen)}
            >
              <Button
                className="group"
                appearance="outlined"
                icon={
                  <ChevronDownIcon className="transform-90 text-slate-500 transition-transform duration-500 group-data-[state=open]:-rotate-180" />
                }
              />
            </CollapsibleCardTrigger>
          </CollapsibleCardHeader>
          {isRemovedFunctionsCardOpen && (
            <CollapsibleCardContent>
              {removedFunctions.map((fn, i) => {
                const isLast = i === removedFunctions.length - 1;

                return (
                  <Link
                    href={pathCreator.function({ envSlug: env.slug, functionSlug: fn.slug })}
                    key={fn.id}
                  >
                    <div
                      className={classNames(
                        defaultLinkStyles,
                        'group flex w-full items-center gap-2 border-slate-200 py-3 pl-6 pr-2 text-sm font-medium hover:bg-slate-100',
                        !isLast && 'border-b'
                      )}
                    >
                      {fn.name}
                      <ArrowRightIcon className="h-3 w-3 -translate-x-3 opacity-0 transition-all group-hover:translate-x-0 group-hover:opacity-100" />
                    </div>
                  </Link>
                );
              })}

              {removedFunctions.length === 0 && (
                <div className="p-2 text-center text-sm text-slate-600">No removed functions</div>
              )}
            </CollapsibleCardContent>
          )}
        </CollapsibleCardItem>
      </CollapsibleCardRoot>
    </div>
  );
}
