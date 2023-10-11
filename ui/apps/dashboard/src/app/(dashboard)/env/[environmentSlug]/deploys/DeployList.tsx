'use client';

import { useSelectedLayoutSegment } from 'next/navigation';

import LoadingIcon from '@/icons/LoadingIcon';
import { useDeploys } from '@/queries/deploys';
import cn from '@/utils/cn';
import { DeployListItem } from './DeployListItem';

type DeployListProps = {
  environmentSlug: string;
};

export default function DeployList({ environmentSlug }: DeployListProps) {
  const [{ data, fetching }] = useDeploys({ environmentSlug });
  const selectedId = useSelectedLayoutSegment();

  const deploys = data?.deploys || [];

  if (fetching || !deploys.length) {
    const shrink = !fetching && deploys.length === 0;
    return (
      <div
        className={cn(
          'flex h-full w-96 items-center justify-center overflow-hidden bg-white shadow transition-all',
          shrink && 'w-0'
        )}
      >
        <LoadingIcon />
      </div>
    );
  }

  // If there are no deploys, we show the onboarding view full width
  if (!deploys.length) {
    return null;
  }

  return (
    <ul className="h-full w-96 shrink-0 divide-y divide-slate-100 overflow-y-scroll bg-white shadow">
      {deploys.map((deploy, index) => {
        if (!deploy) {
          return undefined;
        }

        const isSelected = deploy.id === selectedId;

        return (
          <DeployListItem
            activeFunctionCount={deploy.deployedFunctions.length}
            environmentSlug={environmentSlug}
            createdAt={deploy.createdAt}
            deployID={deploy.id}
            error={deploy.error}
            isSelected={isSelected}
            key={deploy.id}
            removedFunctionCount={deploy.removedFunctions.length}
            status={deploy.status}
          />
        );
      })}
    </ul>
  );
}
