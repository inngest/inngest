import { useMemo } from 'react';
import Link from 'next/link';
import CheckCircleIcon from '@heroicons/react/20/solid/CheckCircleIcon';
import MinusCircleIcon from '@heroicons/react/20/solid/MinusCircleIcon';
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
    <div className={classNames('grid grid-cols-2 border border-slate-300 bg-white', className)}>
      <div className="flex border-b border-r border-slate-300 p-2">
        <CheckCircleIcon className="mr-1 text-green-600" height={20} />
        <h2>Synced Functions ({syncedFunctions.length})</h2>
      </div>
      <div className="flex border-b border-slate-300 p-2">
        <MinusCircleIcon className="mr-1 text-red-600" height={20} />
        <h2>Removed Functions ({removedFunctions.length})</h2>
      </div>

      <div className="border-r border-slate-300">
        {syncedFunctions.map((fn, i) => {
          const isLast = i === syncedFunctions.length - 1;

          return (
            <Link href={`/env/${env.slug}/functions/${encodeURIComponent(fn.slug)}`} key={fn.id}>
              <div
                className={classNames(
                  'border-slate-200 p-2 hover:bg-indigo-50',
                  !isLast && 'border-b'
                )}
              >
                {fn.name}
              </div>
            </Link>
          );
        })}
      </div>

      <div>
        {removedFunctions.map((fn, i) => {
          const isLast = i === removedFunctions.length - 1;

          return (
            <Link href={`/env/${env.slug}/functions/${encodeURIComponent(fn.slug)}`} key={fn.id}>
              <div
                className={classNames(
                  'border-slate-200 p-2 hover:bg-indigo-50',
                  !isLast && 'border-b'
                )}
              >
                {fn.name}
              </div>
            </Link>
          );
        })}

        {removedFunctions.length === 0 && (
          <div className="p-2 text-center text-slate-600">No removed functions</div>
        )}
      </div>
    </div>
  );
}
