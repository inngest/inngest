'use client';

import { Button } from '@inngest/components/Button';

import { StatusTag } from '@/components/Tag/StatusTag';
import LoadingIcon from '@/icons/LoadingIcon';
import CancelledIcon from '@/icons/status-icons/cancelled.svg';
import CompletedIcon from '@/icons/status-icons/completed.svg';
import PausedIcon from '@/icons/status-icons/paused.svg';
import { useFunctionVersions } from '@/queries';
import { defaultTime } from '@/utils/date';
import DateCard from './DateCard';
import VersionCard from './VersionCard';

function Badge({ live }: { live: boolean }) {
  if (live) {
    return (
      <StatusTag size="sm" kind="success" className="ml-4">
        LIVE
      </StatusTag>
    );
  }
  return (
    <StatusTag size="sm" kind="warn" className="ml-4">
      PAUSED
    </StatusTag>
  );
}

function Icon({ live }: { live: boolean }) {
  if (live) {
    return <CompletedIcon className="h-4 w-4" />;
  }
  return <PausedIcon className="h-4 w-4" />;
}

type FunctionVersionsProps = {
  params: {
    environmentSlug: string;
    slug: string;
  };
};

export const runtime = 'nodejs';

export default function FunctionVersions({ params }: FunctionVersionsProps) {
  const functionSlug = decodeURIComponent(params.slug);
  const [{ data: versions, fetching: isFetchingVersions }] = useFunctionVersions({
    environmentSlug: params.environmentSlug,
    functionSlug,
  });

  if (isFetchingVersions) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  const orderedVersions = versions?.sort((a, b) => b.version - a.version);

  return (
    <ul className="h-full flex-1 overflow-y-scroll bg-slate-100 p-6">
      {orderedVersions?.map(
        (v, i) =>
          (
            <VersionCard
              key={v.version}
              status={i === 0 ? 'highlighted' : 'disabled'}
              icon={i === 0 ? <Icon live={!v.validTo} /> : <CancelledIcon className="h-4 w-4" />}
              name={v.version}
              badge={i === 0 ? <Badge live={!v.validTo} /> : undefined}
              dateCards={
                <div className="col-span-2 col-start-4 mx-2 grid grid-cols-2 gap-2">
                  <DateCard
                    description="Deployed at"
                    date={defaultTime(v.validFrom)}
                    variant={i === 0 ? 'dark' : 'light'}
                  />
                  <DateCard
                    description={i === 0 && v.validTo ? 'Paused at' : 'Live until'}
                    date={v.validTo ? defaultTime(v.validTo) : '-'}
                    variant={i === 0 ? 'dark' : 'light'}
                  />
                </div>
              }
              button={
                v.deploy?.id && (
                  <Button
                    appearance={i === 0 ? 'solid' : 'outlined'}
                    href={`/env/${params.environmentSlug}/deploys/${v.deploy.id}`}
                    disabled={!v.deploy?.id}
                    label="View Deploy"
                  />
                )
              }
            />
          ) || 'No versions found'
      )}
    </ul>
  );
}
