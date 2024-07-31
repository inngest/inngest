'use client';

import { Button } from '@inngest/components/Button';
import { useBooleanSearchParam } from '@inngest/components/hooks/useSearchParam';
import { IconFunction } from '@inngest/components/icons/Function';

import { useEnvironment } from '@/components/Environments/environment-context';
import Header, { type HeaderLink } from '@/components/Header/old/Header';
import { FunctionTable } from './FunctionTable';
import { useRows } from './useRows';

export const runtime = 'nodejs';

export default function FunctionListPage() {
  const [archived] = useBooleanSearchParam('archived');

  const { error, isLoading, hasMore, loadMore, rows } = useRows({ archived: archived ?? false });
  if (error) {
    throw error;
  }

  return (
    <>
      <FunctionsHeader />
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

function FunctionsHeader() {
  const env = useEnvironment();
  const [archived] = useBooleanSearchParam('archived');

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
