import { useMemo } from 'react';
import Link from 'next/link';
import ArrowRightIcon from '@heroicons/react/20/solid/ArrowRightIcon';
import type { Function } from '@inngest/components/types/function';
import { classNames } from '@inngest/components/utils/classNames';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';

type Fn = Pick<Function, 'id' | 'name' | 'slug'>;

type Props = {
  className?: string;
  removedFunctions: Fn[];
  syncedFunctions: Fn[];
};

export function FunctionList({ className, removedFunctions, syncedFunctions }: Props) {
  const env = useEnvironment();

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
    <>
      <div className={classNames('mb-4 rounded-lg border border-slate-300 bg-white', className)}>
        <div className="border-b border-r border-slate-300 px-6 py-3 text-sm font-medium text-slate-600">
          <h2>Synced Functions ({syncedFunctions.length})</h2>
        </div>
        <div className="border-r border-slate-300">
          {syncedFunctions.map((fn, i) => {
            const isLast = i === syncedFunctions.length - 1;

            return (
              <Link href={`/env/${env.slug}/functions/${encodeURIComponent(fn.slug)}`} key={fn.id}>
                <div
                  className={classNames(
                    'group flex w-full items-center gap-2 border-slate-200 py-3 pl-6 pr-2 text-sm font-medium text-slate-700 hover:bg-indigo-50  hover:text-indigo-600',
                    !isLast && 'border-b'
                  )}
                >
                  {fn.name}
                  <ArrowRightIcon className="h-3 w-3 -translate-x-3 text-indigo-600 opacity-0 transition-all group-hover:translate-x-0 group-hover:opacity-100" />
                </div>
              </Link>
            );
          })}

          {syncedFunctions.length === 0 && (
            <div className="p-2 text-center text-sm text-slate-600">No synced functions</div>
          )}
        </div>
      </div>
      <div className={classNames('mp-4 rounded-lg border border-slate-300 bg-white', className)}>
        <div className="border-b border-slate-300 px-6 py-3 text-sm font-medium text-slate-600">
          <h2>Removed Functions ({removedFunctions.length})</h2>
        </div>
        <div>
          {removedFunctions.map((fn, i) => {
            const isLast = i === removedFunctions.length - 1;

            return (
              <Link href={`/env/${env.slug}/functions/${encodeURIComponent(fn.slug)}`} key={fn.id}>
                <div
                  className={classNames(
                    'group flex w-full items-center gap-2 border-slate-200 py-3 pl-6 pr-2 text-sm font-medium text-slate-700 hover:bg-indigo-50  hover:text-indigo-600',
                    !isLast && 'border-b'
                  )}
                >
                  {fn.name}
                  <ArrowRightIcon className="h-3 w-3 -translate-x-3 text-indigo-600 opacity-0 transition-all group-hover:translate-x-0 group-hover:opacity-100" />
                </div>
              </Link>
            );
          })}

          {removedFunctions.length === 0 && (
            <div className="p-2 text-center text-sm text-slate-600">No removed functions</div>
          )}
        </div>
      </div>
    </>
  );
}
