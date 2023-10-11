import Link from 'next/link';

import { Alert } from '@/components/Alert';
import DeployStatus from '@/components/Status/DeployStatus';
import { Time } from '@/components/Time';
import ClockIcon from '@/icons/ClockIcon';
import cn from '@/utils/cn';

interface Props {
  environmentSlug: string;
  activeFunctionCount: number | undefined;
  createdAt: string;
  deployID: string;
  error: string | null | undefined;
  isSelected: boolean;
  removedFunctionCount: number | undefined;
  status: string | undefined;
}

export function DeployListItem({
  environmentSlug,
  activeFunctionCount,
  createdAt,
  deployID,
  error,
  isSelected,
  removedFunctionCount,
  status,
}: Props) {
  const classNames = cn(
    'block py-3.5 px-4 hover:bg-slate-100 transition-all w-full',
    isSelected && 'bg-slate-100'
  );

  let functionCountDelta: number | undefined;
  if (removedFunctionCount !== undefined) {
    functionCountDelta = -1 * removedFunctionCount;
  }

  return (
    <li className={classNames} key={deployID}>
      <Link href={`/env/${environmentSlug}/deploys/${deployID}`}>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <DeployStatus status={status || ''} />
            <span className="flex w-40 items-center gap-1 text-sm font-medium leading-none text-slate-600">
              <ClockIcon />
              <Time format="relative" value={new Date(createdAt)} />
            </span>
          </div>

          {!error && activeFunctionCount !== undefined && (
            <FunctionCountBadge count={activeFunctionCount} countDelta={functionCountDelta} />
          )}
        </div>
      </Link>

      {error && (
        <>
          <Alert className="mt-4" severity="error">
            {error}
          </Alert>
        </>
      )}
    </li>
  );
}

interface FunctionCountBadgeProps {
  count: number;
  countDelta: number | undefined;
}

function FunctionCountBadge({ count, countDelta }: FunctionCountBadgeProps) {
  let countDeltaColor = 'text-slate-600';
  if (countDelta !== undefined && countDelta < 0) {
    countDeltaColor = 'text-red-500';
  } else if (countDelta !== undefined && countDelta > 0) {
    countDeltaColor = 'text-teal-500';
  }

  return (
    <span className="flex h-[26px] items-stretch overflow-hidden rounded-full border border-slate-200 bg-slate-100 text-xs">
      <span className="flex items-center bg-white pl-2 pr-1.5 font-medium text-slate-600">
        {count}
      </span>

      {countDelta !== undefined && (
        <span className={`flex items-center pl-1.5 pr-2.5 ${countDeltaColor} font-semibold`}>
          {countDelta >= 0 ? '+' : ''}
          {countDelta}
        </span>
      )}
    </span>
  );
}
