'use client';

import { Button } from '@inngest/components/Button';
import { IconFunction } from '@inngest/components/icons/Function';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import Header, { type HeaderLink } from '@/components/Header/Header';
import { SkeletonCard } from '../apps/AppCard';
import { FunctionTable } from './FunctionTable';
import { useRows } from './useRows';

export default function FunctionListPage({
  searchParams: { archived: isArchived },
}: {
  searchParams: { archived: string };
}) {
  const archived = isArchived === 'true';
  const { error, isLoading, hasMore, loadMore, rows } = useRows({ archived: archived });
  if (error) {
    throw error;
  }
  if (isLoading) {
    return (
      <div className="mb-4 mt-16 flex items-center justify-center">
        <div className="w-full max-w-[1200px]">
          <SkeletonCard />
        </div>
      </div>
    );
  }

  return (
    <>
      <FunctionsHeader archived={archived} />
      <div className="flex min-h-0 flex-1 flex-col divide-y divide-slate-100">
        <FunctionTable rows={rows} />

        {hasMore !== false && (
          <div className="flex w-full justify-center py-2.5">
            <Button
              loading={isLoading}
              appearance="outlined"
              btnAction={loadMore}
              label={isLoading ? 'Loading' : 'Load More'}
            />
          </div>
        )}
      </div>
    </>
  );
}

function FunctionsHeader({ archived }: { archived: boolean }) {
  const env = useEnvironment();

  const navLinks: HeaderLink[] = [
    {
      active: !archived,
      href: `/env/${env.slug}/functions`,
      text: 'Active',
    },
    {
      active: archived,
      href: `/env/${env.slug}/functions?archived=true`,
      text: 'Archived',
    },
  ];

  return (
    <Header
      icon={<IconFunction className="h-5 w-5 text-white" />}
      links={navLinks}
      title="Functions"
    />
  );
}
